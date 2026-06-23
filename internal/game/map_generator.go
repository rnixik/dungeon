package game

import (
	"fmt"
	"math"
	"math/rand"
	"time"
)

// Map generation assembles a fresh map from the predefined templates in a
// template .tmj file:
//
//   - Rooms (class="room") are blocks with thick walls and one or more
//     class="entrance" openings (matched by name). Two are special: "room_start"
//     (where players spawn) and the room containing the demon boss spawn.
//   - Passages (class="passage") are whole corridor tile pieces: "horizontal"
//     and "vertical" straights, four "turn_*" L-bends, a 4-way "crossroad" and
//     four "t_cross_*" T-junctions.
//
// Rooms are laid out on a grid with variable column widths and row heights. A
// street runs in every gap between rooms AND around the whole grid (a perimeter
// ring), so every room is bordered by streets on all four sides. Streets are
// rendered by stamping the whole straight passage pieces end to end (never
// copying single tiles). At each street intersection the matching junction piece
// is stamped: a crossroad in the interior, a t_cross_* where an interior street
// meets the perimeter, and a turn at the four perimeter corners. Rooms are
// connected to the streets through their existing entrances only. A final pass
// turns void tiles touching floor into walls.

const (
	genMargin = 2 // void border around the whole map
	genReach  = 8 // how far an entrance connector reaches into the room interior
)

type roomSide byte

const (
	sideNorth roomSide = 'N'
	sideSouth roomSide = 'S'
	sideEast  roomSide = 'E'
	sideWest  roomSide = 'W'
)

type entrance struct {
	side    roomSide
	off     int
	openLen int
}

type roomTemplate struct {
	name           string
	tx, ty, tw, th int
	entrances      []entrance
	isStart        bool
	isDemon        bool
}

// passagePiece is a corridor tile block plus the detected offsets (within the
// block) of its open channels: vChanOff is the column where the vertical
// (N/S) channel starts, hChanOff the row where the horizontal (W/E) channel
// starts.
type passagePiece struct {
	tx, ty, tw, th     int
	vChanOff, hChanOff int
}

type passageSet struct {
	pieces map[string]passagePiece

	hFloorTop  int // first open row of the horizontal straight's channel
	hChannel   int // horizontal channel height
	vFloorLeft int // first open col of the vertical straight's channel
	vChannel   int // vertical channel width
	cVOff      int // crossroad vertical-channel column offset (canonical gap offset)
	cHOff      int // crossroad horizontal-channel row offset
}

func (p passageSet) piece(name string) passagePiece { return p.pieces[name] }

func generateMap(template *Map, numRooms int, seed int64) (*Map, error) {
	if numRooms < 1 {
		return nil, fmt.Errorf("numRooms must be >= 1, got %d", numRooms)
	}
	if seed == 0 {
		seed = time.Now().UnixNano()
	}
	rng := rand.New(rand.NewSource(seed))

	templates, err := extractRoomTemplates(template)
	if err != nil {
		return nil, err
	}
	pass, err := extractPassages(template)
	if err != nil {
		return nil, err
	}
	floorGid, wallGid, err := pickCorridorGids(template, pass)
	if err != nil {
		return nil, err
	}

	place := chooseRooms(templates, numRooms, rng)
	cols, rows := gridDims(numRooms)

	// Variable column widths and row heights (cell i in row-major order). The gap
	// between rooms is exactly one corridor lane wide: a vertical lane is as wide
	// as the vertical straight piece, a horizontal lane as tall as the horizontal
	// straight piece. The larger crossroad/T/turn pieces stamped at intersections
	// overhang into the rooms, but their void corners are skipped so nothing is
	// erased.
	gapW, gapH := pass.piece("vertical").tw, pass.piece("horizontal").th
	colWidth := make([]int, cols)
	rowHeight := make([]int, rows)
	for i, t := range place {
		r, c := i/cols, i%cols
		colWidth[c] = max(colWidth[c], t.tw)
		rowHeight[r] = max(rowHeight[r], t.th)
	}

	// Gaps surround every room (perimeter ring): cols+1 vertical gaps, rows+1
	// horizontal gaps. vGapX[g] is the left edge of vertical gap g.
	vGapX := make([]int, cols+1)
	hGapY := make([]int, rows+1)
	roomX := make([]int, cols)
	roomY := make([]int, rows)
	vGapX[0] = genMargin
	for c := 0; c < cols; c++ {
		roomX[c] = vGapX[c] + gapW
		vGapX[c+1] = roomX[c] + colWidth[c]
	}
	hGapY[0] = genMargin
	for r := 0; r < rows; r++ {
		roomY[r] = hGapY[r] + gapH
		hGapY[r+1] = roomY[r] + rowHeight[r]
	}
	canvasW := vGapX[cols] + gapW + genMargin
	canvasH := hGapY[rows] + gapH + genMargin

	// Channel position of a lane: aligned to the straight piece's own channel
	// within the (narrow) gap. Junction pieces are placed so their channels line
	// up with this.
	vChanLeft := func(g int) int { return vGapX[g] + pass.vFloorLeft }
	hChanTop := func(g int) int { return hGapY[g] + pass.hFloorTop }

	out := newBlankMap(template, canvasW, canvasH)
	floor := out.tileLayerData("floor")
	walls := out.tileLayerData("walls")
	if floor == nil || walls == nil {
		return nil, fmt.Errorf("template map is missing a floor or walls layer")
	}
	// carve opens a connector corridor: it clears walls and fills only empty
	// tiles with the corridor floor, leaving existing room/passage floor intact so
	// textures are preserved.
	carve := func(x0, x1, y0, y1 int) {
		if x0 > x1 {
			x0, x1 = x1, x0
		}
		if y0 > y1 {
			y0, y1 = y1, y0
		}
		for y := max(0, y0); y <= min(canvasH-1, y1); y++ {
			for x := max(0, x0); x <= min(canvasW-1, x1); x++ {
				i := x + y*canvasW
				walls[i] = 0
				if floor[i] == 0 {
					floor[i] = floorGid
				}
			}
		}
	}

	// originOf places a room within its cell flush against the lane(s) its
	// entrances face, so the entrance meets the corridor directly (no gap). Rooms
	// default to the top-left of the cell; an east-only entrance pushes it right,
	// a south-only entrance pushes it down.
	originOf := func(i int) (int, int) {
		t := place[i]
		r, c := i/cols, i%cols
		ox, oy := roomX[c], roomY[r]
		var hasN, hasS, hasE, hasW bool
		for _, e := range t.entrances {
			switch e.side {
			case sideNorth:
				hasN = true
			case sideSouth:
				hasS = true
			case sideEast:
				hasE = true
			case sideWest:
				hasW = true
			}
		}
		if hasE && !hasW {
			ox = vGapX[c+1] - t.tw
		}
		if hasS && !hasN {
			oy = hGapY[r+1] - t.th
		}
		return ox, oy
	}

	objects := make([]MapObject, 0)
	spawns := make([]MapObject, 0)
	nextID := 1

	// --- Stamp rooms ---
	for i, t := range place {
		ox, oy := originOf(i)
		stampRoom(out, template, t, ox, oy)
		objects = appendRoomObjects(objects, template, t, ox, oy, &nextID)
		spawns = appendRoomSpawns(spawns, template, t, ox, oy, &nextID)
		if t.isStart {
			out.spawnX = (ox + t.tw/2) * out.TileWidth
			out.spawnY = (oy + t.th/2) * out.TileHeight
		}
	}

	stamp := func(name string, x, y int) {
		pc := pass.piece(name)
		copyBlock(out, template, pc.tx, pc.ty, pc.tw, pc.th, x, y)
	}

	// --- Stamp a T-junction branching off the street toward each entrance. This
	// happens BEFORE the streets are laid so that re-stamping the streets and
	// junctions afterwards repairs any street tiles a T overlapped; the branch
	// stub toward the room survives. A T is only placed where it fits between the
	// room's bounding streets. ---
	for i, t := range place {
		r, c := i/cols, i%cols
		ox, oy := originOf(i)
		fitsCols := func(x0, w int) bool {
			return x0 > vChanLeft(c)+pass.vChannel-1 && x0+w-1 < vChanLeft(c+1)
		}
		fitsRows := func(y0, h int) bool {
			return y0 > hChanTop(r)+pass.hChannel-1 && y0+h-1 < hChanTop(r+1)
		}
		for _, e := range t.entrances {
			centerX := ox + e.off + e.openLen/2
			centerY := oy + e.off + e.openLen/2
			switch e.side {
			case sideNorth:
				td := pass.piece("t_cross_down")
				if tx := centerX - (td.vChanOff + pass.vChannel/2); fitsCols(tx, td.tw) {
					stamp("t_cross_down", tx, hChanTop(r)-td.hChanOff)
				}
			case sideSouth:
				tu := pass.piece("t_cross_up")
				if tx := centerX - (tu.vChanOff + pass.vChannel/2); fitsCols(tx, tu.tw) {
					stamp("t_cross_up", tx, hChanTop(r+1)-tu.hChanOff)
				}
			case sideEast:
				tl := pass.piece("t_cross_left")
				if ty := centerY - (tl.hChanOff + pass.hChannel/2); fitsRows(ty, tl.th) {
					stamp("t_cross_left", vChanLeft(c+1)-tl.vChanOff, ty)
				}
			case sideWest:
				tr := pass.piece("t_cross_right")
				if ty := centerY - (tr.hChanOff + pass.hChannel/2); fitsRows(ty, tr.th) {
					stamp("t_cross_right", vChanLeft(c)-tr.vChanOff, ty)
				}
			}
		}
	}

	// --- Streets from whole passage pieces, spanning the perimeter bounds ---
	streetX0, streetX1 := vChanLeft(0), vChanLeft(cols)+pass.vChannel-1
	streetY0, streetY1 := hChanTop(0), hChanTop(rows)+pass.hChannel-1
	for g := 0; g <= cols; g++ {
		extrudeVertical(out, template, pass, vChanLeft(g), streetY0, streetY1)
	}
	for g := 0; g <= rows; g++ {
		extrudeHorizontal(out, template, pass, hChanTop(g), streetX0, streetX1)
	}

	// --- Junction piece at every street intersection ---
	for gr := 0; gr <= rows; gr++ {
		for gc := 0; gc <= cols; gc++ {
			pc := pass.piece(junctionName(gc, gr, cols, rows))
			copyBlock(out, template, pc.tx, pc.ty, pc.tw, pc.th,
				vChanLeft(gc)-pc.vChanOff, hChanTop(gr)-pc.hChanOff)
		}
	}

	// --- Carve the branch from each entrance to its street. The carve clears
	// walls and fills only empty tiles (preserving room/passage floor), and
	// always runs so connectivity holds regardless of the T placement. ---
	for i, t := range place {
		r, c := i/cols, i%cols
		ox, oy := originOf(i)
		for _, e := range t.entrances {
			a, b := ox+e.off, ox+e.off+e.openLen-1
			ay, by := oy+e.off, oy+e.off+e.openLen-1
			centerX := ox + e.off + e.openLen/2
			centerY := oy + e.off + e.openLen/2
			vLo := min(a, centerX-pass.vChannel/2)
			vHi := max(b, centerX-pass.vChannel/2+pass.vChannel-1)
			hLo := min(ay, centerY-pass.hChannel/2)
			hHi := max(by, centerY-pass.hChannel/2+pass.hChannel-1)
			switch e.side {
			case sideNorth:
				carve(vLo, vHi, hChanTop(r), oy+genReach)
			case sideSouth:
				carve(vLo, vHi, oy+t.th-genReach, hChanTop(r+1)+pass.hChannel-1)
			case sideEast:
				carve(ox+t.tw-genReach, vChanLeft(c+1)+pass.vChannel-1, hLo, hHi)
			case sideWest:
				carve(vChanLeft(c), ox+genReach, hLo, hHi)
			}
		}
	}

	autoWall(floor, walls, canvasW, canvasH, wallGid)

	out.setObjectLayer("objects", objects)
	out.setObjectLayer("spawns", spawns)

	return out, nil
}

// junctionName returns which passage piece sits at street intersection (gc,gr):
// turns at the four corners, t_cross_* where an interior street meets the
// perimeter, and a crossroad in the interior.
func junctionName(gc, gr, cols, rows int) string {
	left, right := gc == 0, gc == cols
	top, bottom := gr == 0, gr == rows
	switch {
	case left && top:
		return "turn_left_upper" // opens E,S
	case right && top:
		return "turn_right_upper" // opens W,S
	case left && bottom:
		return "turn_left_bottom" // opens E,N
	case right && bottom:
		return "turn_right_bottom" // opens W,N
	case top:
		return "t_cross_down" // opens W,E,S
	case bottom:
		return "t_cross_up" // opens W,E,N
	case left:
		return "t_cross_right" // opens N,S,E
	case right:
		return "t_cross_left" // opens N,S,W
	default:
		return "crossroad"
	}
}

// gridDims returns a near-square columns x rows grid that holds n rooms.
func gridDims(n int) (cols, rows int) {
	cols = int(math.Ceil(math.Sqrt(float64(n))))
	rows = int(math.Ceil(float64(n) / float64(cols)))
	return cols, rows
}

// chooseRooms selects which templates to place: room_start first, the demon room
// last, every other template at least once, then random fills (never duplicating
// the two special rooms).
func chooseRooms(templates []roomTemplate, n int, rng *rand.Rand) []roomTemplate {
	var start, demon *roomTemplate
	var normal []roomTemplate
	for i := range templates {
		switch {
		case templates[i].isStart:
			start = &templates[i]
		case templates[i].isDemon:
			demon = &templates[i]
		default:
			normal = append(normal, templates[i])
		}
	}
	if start == nil {
		start = &templates[0]
		normal = templates[1:]
	}

	out := make([]roomTemplate, 0, n)
	out = append(out, *start)
	reserve := 0
	if demon != nil && n >= 2 {
		reserve = 1
	}
	for i := 0; i < len(normal) && len(out) < n-reserve; i++ {
		out = append(out, normal[i])
	}
	for len(out) < n-reserve && len(normal) > 0 {
		out = append(out, normal[rng.Intn(len(normal))])
	}
	if reserve == 1 {
		out = append(out, *demon)
	}
	return out
}

// extractRoomTemplates reads every class="room", its class="entrance" openings,
// and whether it is the start room or contains the demon spawn.
func extractRoomTemplates(template *Map) ([]roomTemplate, error) {
	objectsLayer := template.objectLayer("objects")
	if objectsLayer == nil {
		return nil, fmt.Errorf("template map has no objects layer")
	}
	spawnLayer := template.objectLayer("spawns")
	ts := template.TileWidth

	entrancesByName := make(map[string][]MapObject)
	var rooms []MapObject
	for _, o := range objectsLayer.Objects {
		switch o.Type {
		case "room":
			rooms = append(rooms, o)
		case "entrance":
			entrancesByName[o.Name] = append(entrancesByName[o.Name], o)
		}
	}
	if len(rooms) == 0 {
		return nil, fmt.Errorf("template map has no class=room objects")
	}

	var templates []roomTemplate
	for _, r := range rooms {
		rt := roomTemplate{
			name:    r.Name,
			tx:      int(r.X) / ts,
			ty:      int(r.Y) / ts,
			tw:      int(r.Width) / ts,
			th:      int(r.Height) / ts,
			isStart: r.Name == "room_start",
		}
		for _, e := range entrancesByName[r.Name] {
			ex, ey := int(e.X)/ts, int(e.Y)/ts
			ew, eh := int(e.Width)/ts, int(e.Height)/ts
			var en entrance
			switch {
			case ex <= rt.tx:
				en = entrance{sideWest, ey - rt.ty, eh}
			case ex+ew >= rt.tx+rt.tw:
				en = entrance{sideEast, ey - rt.ty, eh}
			case ey <= rt.ty:
				en = entrance{sideNorth, ex - rt.tx, ew}
			default:
				en = entrance{sideSouth, ex - rt.tx, ew}
			}
			rt.entrances = append(rt.entrances, en)
		}
		if len(rt.entrances) == 0 {
			return nil, fmt.Errorf("room %q has no entrance objects", r.Name)
		}
		if spawnLayer != nil {
			rx0, ry0 := float64(rt.tx*ts), float64(rt.ty*ts)
			rx1, ry1 := float64((rt.tx+rt.tw)*ts), float64((rt.ty+rt.th)*ts)
			for _, o := range spawnLayer.Objects {
				if o.Name != "demon" {
					continue
				}
				cx, cy := o.X+o.Width/2, o.Y+o.Height/2
				if cx >= rx0 && cx < rx1 && cy >= ry0 && cy < ry1 {
					rt.isDemon = true
					break
				}
			}
		}
		templates = append(templates, rt)
	}
	return templates, nil
}

// extractPassages reads every corridor piece and detects its channel geometry.
func extractPassages(template *Map) (passageSet, error) {
	objectsLayer := template.objectLayer("objects")
	if objectsLayer == nil {
		return passageSet{}, fmt.Errorf("template map has no objects layer")
	}
	walls := template.tileLayerData("walls")
	floor := template.tileLayerData("floor")
	if walls == nil || floor == nil {
		return passageSet{}, fmt.Errorf("template map is missing a floor or walls layer")
	}
	open := func(x, y int) bool { return walls[x+y*template.Width] == 0 && floor[x+y*template.Width] != 0 }
	firstRun := func(get func(i int) bool, n int) (off, length int) {
		for i := 0; i < n; i++ {
			if get(i) {
				if length == 0 {
					off = i
				}
				length++
			} else if length > 0 {
				break
			}
		}
		return
	}

	ts := template.TileWidth
	ps := passageSet{pieces: make(map[string]passagePiece)}
	for _, o := range objectsLayer.Objects {
		if o.Type != "passage" {
			continue
		}
		pc := passagePiece{tx: int(o.X) / ts, ty: int(o.Y) / ts, tw: int(o.Width) / ts, th: int(o.Height) / ts}
		// Vertical channel: an open run on the top edge, else the bottom edge.
		if off, n := firstRun(func(i int) bool { return open(pc.tx+i, pc.ty) }, pc.tw); n > 0 {
			pc.vChanOff = off
		} else if off, n := firstRun(func(i int) bool { return open(pc.tx+i, pc.ty+pc.th-1) }, pc.tw); n > 0 {
			pc.vChanOff = off
		}
		// Horizontal channel: an open run on the left edge, else the right edge.
		if off, n := firstRun(func(i int) bool { return open(pc.tx, pc.ty+i) }, pc.th); n > 0 {
			pc.hChanOff = off
		} else if off, n := firstRun(func(i int) bool { return open(pc.tx+pc.tw-1, pc.ty+i) }, pc.th); n > 0 {
			pc.hChanOff = off
		}
		ps.pieces[o.Name] = pc
	}

	need := []string{"horizontal", "vertical", "crossroad",
		"turn_right_upper", "turn_right_bottom", "turn_left_upper", "turn_left_bottom",
		"t_cross_down", "t_cross_up", "t_cross_left", "t_cross_right"}
	for _, name := range need {
		if _, ok := ps.pieces[name]; !ok {
			return passageSet{}, fmt.Errorf("template map is missing passage %q", name)
		}
	}

	h := ps.pieces["horizontal"]
	v := ps.pieces["vertical"]
	cr := ps.pieces["crossroad"]
	ps.hFloorTop, ps.hChannel = firstRun(func(i int) bool { return open(h.tx+h.tw/2, h.ty+i) }, h.th)
	ps.vFloorLeft, ps.vChannel = firstRun(func(i int) bool { return open(v.tx+i, v.ty+v.th/2) }, v.tw)
	ps.cVOff, ps.cHOff = cr.vChanOff, cr.hChanOff
	if ps.hChannel == 0 || ps.vChannel == 0 {
		return passageSet{}, fmt.Errorf("could not detect passage channels")
	}
	return ps, nil
}

// extrudeHorizontal stamps whole horizontal straight pieces end to end along a
// run, aligning the channel to rows starting at channelTop.
func extrudeHorizontal(out, template *Map, p passageSet, channelTop, x0, x1 int) {
	h := p.piece("horizontal")
	dstTop := channelTop - p.hFloorTop
	for x := x0; x <= x1; x += h.tw {
		copyBlock(out, template, h.tx, h.ty, min(h.tw, x1-x+1), h.th, x, dstTop)
	}
}

// extrudeVertical stamps whole vertical straight pieces end to end along a run,
// aligning the channel to cols starting at channelLeft.
func extrudeVertical(out, template *Map, p passageSet, channelLeft, y0, y1 int) {
	v := p.piece("vertical")
	dstLeft := channelLeft - p.vFloorLeft
	for y := y0; y <= y1; y += v.th {
		copyBlock(out, template, v.tx, v.ty, v.tw, min(v.th, y1-y+1), dstLeft, y)
	}
}

// copyBlock copies a w x h tile block from the template into the output map at
// the given tile offset, across every tile layer. Void (empty) source tiles are
// skipped so a piece's transparent areas (e.g. a crossroad's void corners) never
// erase tiles already placed underneath (e.g. a room corner nestled into them).
func copyBlock(out, template *Map, srcX, srcY, w, h, dstX, dstY int) {
	type layerPair struct {
		dst []int
		src []int
	}
	pairs := make([]layerPair, 0, len(out.Layers))
	for li := range out.Layers {
		l := &out.Layers[li]
		if l.Type != "tilelayer" {
			continue
		}
		if src := template.tileLayerData(l.Name); src != nil {
			pairs = append(pairs, layerPair{dst: l.Data, src: src})
		}
	}
	for dy := 0; dy < h; dy++ {
		ny, sy := dstY+dy, srcY+dy
		if ny < 0 || ny >= out.Height || sy < 0 || sy >= template.Height {
			continue
		}
		for dx := 0; dx < w; dx++ {
			nx, sx := dstX+dx, srcX+dx
			if nx < 0 || nx >= out.Width || sx < 0 || sx >= template.Width {
				continue
			}
			// A tile occupies the cell if any layer has a non-void tile there;
			// only copy the cell when the source actually has content, and within
			// the cell copy every layer (so multi-layer tiles stay aligned).
			occupied := false
			for _, p := range pairs {
				if p.src[sx+sy*template.Width] != 0 {
					occupied = true
					break
				}
			}
			if !occupied {
				continue
			}
			for _, p := range pairs {
				p.dst[nx+ny*out.Width] = p.src[sx+sy*template.Width]
			}
		}
	}
}

func stampRoom(out, template *Map, t roomTemplate, ox, oy int) {
	copyBlock(out, template, t.tx, t.ty, t.tw, t.th, ox, oy)
}

func appendRoomObjects(dst []MapObject, template *Map, t roomTemplate, ox, oy int, nextID *int) []MapObject {
	return appendTranslatedObjects(dst, template, "objects", t, ox, oy, nextID, func(o MapObject) bool {
		return o.Type != "room" && o.Type != "entrance" && o.Type != "passage"
	})
}

func appendRoomSpawns(dst []MapObject, template *Map, t roomTemplate, ox, oy int, nextID *int) []MapObject {
	return appendTranslatedObjects(dst, template, "spawns", t, ox, oy, nextID, func(o MapObject) bool { return true })
}

// appendTranslatedObjects copies objects whose centre is inside the room
// rectangle, translating them to the placed position with a fresh id.
func appendTranslatedObjects(dst []MapObject, template *Map, layerName string, t roomTemplate, ox, oy int, nextID *int, keep func(MapObject) bool) []MapObject {
	layer := template.objectLayer(layerName)
	if layer == nil {
		return dst
	}
	ts := float64(template.TileWidth)
	dxPix := float64(ox-t.tx) * ts
	dyPix := float64(oy-t.ty) * ts
	rx0, ry0 := float64(t.tx)*ts, float64(t.ty)*ts
	rx1, ry1 := float64(t.tx+t.tw)*ts, float64(t.ty+t.th)*ts
	for _, o := range layer.Objects {
		cx, cy := o.X+o.Width/2, o.Y+o.Height/2
		if cx < rx0 || cx >= rx1 || cy < ry0 || cy >= ry1 || !keep(o) {
			continue
		}
		o.X += dxPix
		o.Y += dyPix
		o.Id = *nextID
		*nextID++
		dst = append(dst, o)
	}
	return dst
}

// pickCorridorGids chooses the floor and wall tiles used for the carved entrance
// connectors. Both are sampled from the passage pieces so connectors match the
// corridors they join: the most common floor tile and the most common
// light-absorbing (collidable) wall tile within the passage blocks.
func pickCorridorGids(template *Map, pass passageSet) (floorGid, wallGid int, err error) {
	absorbs := make(map[int]bool)
	for _, set := range template.Tilesets {
		for _, tile := range set.Tiles {
			for _, pr := range tile.Properties {
				if pr.Name == "absorbs_light" {
					if v, ok := pr.Value.(bool); ok && v {
						absorbs[set.Firstgid+tile.Id] = true
					}
				}
			}
		}
	}
	floorLayer := template.tileLayerData("floor")
	wallLayer := template.tileLayerData("walls")
	if floorLayer == nil || wallLayer == nil {
		return 0, 0, fmt.Errorf("template map is missing a floor or walls layer")
	}

	floorCounts := make(map[int]int)
	wallCounts := make(map[int]int)
	for _, pc := range pass.pieces {
		for y := pc.ty; y < pc.ty+pc.th; y++ {
			for x := pc.tx; x < pc.tx+pc.tw; x++ {
				i := x + y*template.Width
				if g := floorLayer[i]; g != 0 {
					floorCounts[g]++
				}
				if g := wallLayer[i]; absorbs[g] {
					wallCounts[g]++
				}
			}
		}
	}
	mostCommon := func(m map[int]int) (gid, n int) {
		for g, c := range m {
			if c > n {
				gid, n = g, c
			}
		}
		return
	}
	var fn, wn int
	floorGid, fn = mostCommon(floorCounts)
	wallGid, wn = mostCommon(wallCounts)
	if fn == 0 {
		return 0, 0, fmt.Errorf("no floor tiles found in passage pieces")
	}
	if wn == 0 {
		return 0, 0, fmt.Errorf("no light-absorbing wall tiles found in passage pieces")
	}
	return floorGid, wallGid, nil
}

// autoWall turns every void tile 8-adjacent to a floor tile into a wall.
func autoWall(floor, walls []int, w, h, wallGid int) {
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			i := x + y*w
			if walls[i] != 0 || floor[i] != 0 {
				continue
			}
			adjacent := false
			for dy := -1; dy <= 1 && !adjacent; dy++ {
				for dx := -1; dx <= 1; dx++ {
					if dx == 0 && dy == 0 {
						continue
					}
					nx, ny := x+dx, y+dy
					if nx < 0 || nx >= w || ny < 0 || ny >= h {
						continue
					}
					if floor[nx+ny*w] != 0 {
						adjacent = true
						break
					}
				}
			}
			if adjacent {
				walls[i] = wallGid
			}
		}
	}
}

func newBlankMap(template *Map, w, h int) *Map {
	out := &Map{
		Infinite:    false,
		Width:       w,
		Height:      h,
		TileWidth:   template.TileWidth,
		TileHeight:  template.TileHeight,
		Tilesets:    template.Tilesets,
		Orientation: template.Orientation,
		RenderOrder: template.RenderOrder,
		Type:        template.Type,
		Version:     template.Version,
	}
	out.Layers = make([]MapLayer, 0, len(template.Layers))
	for _, src := range template.Layers {
		layer := MapLayer{
			Name: src.Name, Type: src.Type, Visible: src.Visible,
			ID: src.ID, Opacity: src.Opacity, Draworder: src.Draworder,
		}
		if src.Type == "tilelayer" {
			layer.Width, layer.Height = w, h
			layer.Data = make([]int, w*h)
		} else {
			layer.Objects = make([]MapObject, 0)
		}
		out.Layers = append(out.Layers, layer)
	}
	return out
}

func (m *Map) tileLayerData(name string) []int {
	for i := range m.Layers {
		if m.Layers[i].Name == name && m.Layers[i].Type == "tilelayer" {
			return m.Layers[i].Data
		}
	}
	return nil
}

func (m *Map) objectLayer(name string) *MapLayer {
	for i := range m.Layers {
		if m.Layers[i].Name == name && m.Layers[i].Type == "objectgroup" {
			return &m.Layers[i]
		}
	}
	return nil
}

func (m *Map) setObjectLayer(name string, objects []MapObject) {
	if l := m.objectLayer(name); l != nil {
		l.Objects = objects
	}
}
