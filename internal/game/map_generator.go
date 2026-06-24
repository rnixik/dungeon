package game

import (
	"fmt"
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
// Layout is a single horizontal "spine" corridor with rooms branching off it,
// not a grid. The spine is tiled from whole horizontal straight pieces. Each room
// hangs off the spine through one of its real entrances:
//
//   - a north door -> room sits below the spine, reached by a t_cross_down + a
//     vertical riser straight into the room's top;
//   - a south door -> room sits above the spine, reached by a t_cross_up + a riser
//     into the room's bottom;
//   - an east/west door -> a vertical riser drops off the spine and a turn piece
//     bends it horizontally into the room's side.
//
// The spine's two ends are capped with turn pieces (so no channel mouth is left
// open to the void); interior branches use t_cross pieces. Every piece carries
// its own walls, so corridors are self-sealing and there is no fill/auto-wall
// step: any door not used by a corridor is closed with the corridor wall tile.
// Piece stamping skips void tiles so a piece's transparent corners never erase a
// neighbour.

const (
	genMargin    = 2  // void border around the whole map
	genGap       = 4  // horizontal gap between rooms along the spine
	genUpRiser   = 16 // clearance above the spine for south-door rooms
	genDownRiser = 9  // clearance below the spine before north-door rooms
	genStub      = 3  // short horizontal stub into an east/west room
	genTRight    = 18 // how far a spine junction reaches right of its channel
	genReach     = 10 // how far a connected door is carved through the room wall
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
}

func (p passageSet) piece(name string) passagePiece { return p.pieces[name] }

// slot is a placed room plus the geometry of the single corridor branch that
// connects it to the spine.
type slot struct {
	t          roomTemplate
	pe         entrance // the entrance the corridor uses
	dir        byte     // 'U' room above the spine, 'D' room below
	roomX      int
	roomY      int
	cx         int    // left column of the branch's vertical channel
	ew         bool   // true for an east/west door reached via a turn
	goWest     bool   // east door: corridor bends west into the room
	cornerName string // turn piece used for the east/west bend
	ry         int    // row of the horizontal stub's channel (east/west only)
}

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
	n := len(place)

	// --- Pass 1: horizontal layout. Walk left to right assigning each room an x
	// position and the column of its branch's vertical channel. ---
	slots := make([]slot, n)
	curX := genMargin + genGap
	// Consecutive branch junctions must be at least a t_cross apart so their
	// pieces never overlap and sever the spine.
	minGap := pass.piece("t_cross_down").tw
	prevCx := 0
	for i, t := range place {
		pe := primaryEntrance(t)
		s := slot{t: t, pe: pe}
		switch pe.side {
		case sideNorth:
			s.dir = 'D'
		case sideSouth:
			s.dir = 'U'
		case sideEast:
			s.dir, s.ew, s.goWest, s.cornerName = 'D', true, true, "turn_right_bottom"
		case sideWest:
			s.dir, s.ew, s.goWest, s.cornerName = 'D', true, false, "turn_left_bottom"
		}

		switch {
		case !s.ew:
			s.roomX = curX
			s.cx = s.roomX + pe.off - (pass.vChannel-pe.openLen)/2
			right := max(s.roomX+t.tw, s.cx+genTRight)
			curX = right + genGap
		case s.goWest: // east door: room first, riser to its right
			s.roomX = curX
			cornerX := s.roomX + t.tw + genStub
			s.cx = cornerX + pass.piece(s.cornerName).vChanOff
			curX = s.cx + genTRight + genGap
		default: // west door: riser first, room to its right
			s.cx = curX + genTRight
			corner := pass.piece(s.cornerName)
			cornerX := s.cx - corner.vChanOff
			s.roomX = cornerX + corner.tw + genStub
			curX = s.roomX + t.tw + genGap
		}
		if i > 0 {
			if d := prevCx + minGap - s.cx; d > 0 {
				s.roomX += d
				s.cx += d
				curX += d
			}
		}
		prevCx = s.cx
		slots[i] = s
	}
	canvasW := curX + genMargin

	// --- Pass 2: vertical layout. Bands above and below the spine. ---
	aboveMaxH := 0
	for _, s := range slots {
		if s.dir == 'U' {
			aboveMaxH = max(aboveMaxH, s.t.th)
		}
	}
	aboveBandBottom := genMargin + aboveMaxH
	spineY := aboveBandBottom + genUpRiser
	belowBandTop := spineY + pass.piece("t_cross_down").th + genDownRiser

	maxBottom := belowBandTop
	for i := range slots {
		s := &slots[i]
		if s.dir == 'U' {
			s.roomY = aboveBandBottom - s.t.th
			continue
		}
		s.roomY = belowBandTop
		maxBottom = max(maxBottom, s.roomY+s.t.th)
		if s.ew {
			s.ry = s.roomY + s.pe.off - (pass.hChannel-s.pe.openLen)/2
			corner := pass.piece(s.cornerName)
			maxBottom = max(maxBottom, s.ry-corner.hChanOff+corner.th)
		}
	}
	canvasH := maxBottom + genMargin

	out := newBlankMap(template, canvasW, canvasH)
	floor := out.tileLayerData("floor")
	walls := out.tileLayerData("walls")
	if floor == nil || walls == nil {
		return nil, fmt.Errorf("template map is missing a floor or walls layer")
	}

	stamp := func(name string, dstX, dstY int) passagePiece {
		p := pass.piece(name)
		copyBlock(out, template, p.tx, p.ty, p.tw, p.th, dstX, dstY)
		return p
	}
	// stampJunction places a piece aligning its vertical channel to column cx and
	// its horizontal channel to row ry.
	stampJunction := func(name string, cx, ry int) passagePiece {
		p := pass.piece(name)
		copyBlock(out, template, p.tx, p.ty, p.tw, p.th, cx-p.vChanOff, ry-p.hChanOff)
		return p
	}
	tileV := func(cx, y0, y1 int) {
		if y1 < y0 {
			return
		}
		v := pass.piece("vertical")
		dstX := cx - v.vChanOff
		for y := y0; y <= y1; y += v.th {
			copyBlock(out, template, v.tx, v.ty, v.tw, min(v.th, y1-y+1), dstX, y)
		}
	}
	tileH := func(ry, x0, x1 int) {
		if x1 < x0 {
			return
		}
		h := pass.piece("horizontal")
		dstY := ry - h.hChanOff
		for x := x0; x <= x1; x += h.tw {
			copyBlock(out, template, h.tx, h.ty, min(h.tw, x1-x+1), h.th, x, dstY)
		}
	}
	// carveThroat opens a connected door inward through the room's wall ring: a
	// template room's door is a stub walled off from its interior, so clearing the
	// wall (the floor already exists underneath) joins the door to the room.
	carveThroat := func(rx, ry int, t roomTemplate, e entrance) {
		clear := func(x, y int) {
			if x >= 0 && x < canvasW && y >= 0 && y < canvasH {
				i := x + y*canvasW
				walls[i] = 0
				if floor[i] == 0 {
					floor[i] = floorGid
				}
			}
		}
		for k := 0; k < e.openLen; k++ {
			for d := 0; d <= genReach; d++ {
				switch e.side {
				case sideNorth:
					clear(rx+e.off+k, ry+d)
				case sideSouth:
					clear(rx+e.off+k, ry+t.th-1-d)
				case sideWest:
					clear(rx+d, ry+e.off+k)
				case sideEast:
					clear(rx+t.tw-1-d, ry+e.off+k)
				}
			}
		}
	}
	seal := func(rx, ry int, t roomTemplate, e entrance) {
		set := func(x, y int) {
			if x >= 0 && x < canvasW && y >= 0 && y < canvasH {
				walls[x+y*canvasW] = wallGid
			}
		}
		switch e.side {
		case sideNorth:
			for k := 0; k < e.openLen; k++ {
				set(rx+e.off+k, ry)
			}
		case sideSouth:
			for k := 0; k < e.openLen; k++ {
				set(rx+e.off+k, ry+t.th-1)
			}
		case sideWest:
			for k := 0; k < e.openLen; k++ {
				set(rx, ry+e.off+k)
			}
		case sideEast:
			for k := 0; k < e.openLen; k++ {
				set(rx+t.tw-1, ry+e.off+k)
			}
		}
	}

	// --- Spine straights between the two end caps ---
	if n >= 2 {
		lp := pass.piece(junctionName(slots[0].dir, 0, n))
		rp := pass.piece(junctionName(slots[n-1].dir, n-1, n))
		spineX0 := slots[0].cx - lp.vChanOff + lp.tw
		spineX1 := slots[n-1].cx - rp.vChanOff
		tileH(spineY, spineX0, spineX1-1)
	}

	objects := make([]MapObject, 0)
	spawns := make([]MapObject, 0)
	nextID := 1

	for i := range slots {
		s := &slots[i]
		stampRoom(out, template, s.t, s.roomX, s.roomY)
		objects = appendRoomObjects(objects, template, s.t, s.roomX, s.roomY, &nextID)
		spawns = appendRoomSpawns(spawns, template, s.t, s.roomX, s.roomY, &nextID)
		out.roomCenters = append(out.roomCenters, [2]int{s.roomX + s.t.tw/2, s.roomY + s.t.th/2})
		if s.t.isStart {
			out.spawnX = (s.roomX + s.t.tw/2) * out.TileWidth
			out.spawnY = (s.roomY + s.t.th/2) * out.TileHeight
		}
		if s.t.isDemon {
			out.demonRoomCount++
		}

		// A lone room has no spine: just seal all its doors.
		if n >= 2 {
			carveThroat(s.roomX, s.roomY, s.t, s.pe)
			jp := stampJunction(junctionName(s.dir, i, n), s.cx, spineY)
			switch {
			case s.dir == 'U':
				top := spineY - jp.hChanOff
				tileV(s.cx, s.roomY+s.t.th, top-1)
			case s.ew:
				below := spineY - jp.hChanOff + jp.th
				corner := pass.piece(s.cornerName)
				cornerX := s.cx - corner.vChanOff
				cornerY := s.ry - corner.hChanOff
				tileV(s.cx, below, cornerY-1)
				stamp(s.cornerName, cornerX, cornerY)
				if s.goWest {
					tileH(s.ry, s.roomX+s.t.tw, cornerX-1)
				} else {
					tileH(s.ry, cornerX+corner.tw, s.roomX-1)
				}
			default: // 'D' north door
				below := spineY - jp.hChanOff + jp.th
				tileV(s.cx, below, s.roomY-1)
			}
		}

		for _, e := range s.t.entrances {
			if n >= 2 && e == s.pe {
				continue
			}
			seal(s.roomX, s.roomY, s.t, e)
		}
	}

	out.setObjectLayer("objects", objects)
	out.setObjectLayer("spawns", spawns)

	return out, nil
}

// primaryEntrance picks the door a room hangs from, preferring vertical doors
// (which need only a riser) over side doors (which need an extra turn).
func primaryEntrance(t roomTemplate) entrance {
	for _, want := range []roomSide{sideNorth, sideSouth, sideEast, sideWest} {
		for _, e := range t.entrances {
			if e.side == want {
				return e
			}
		}
	}
	return t.entrances[0]
}

// junctionName returns the piece that taps the spine for room i of n: a turn at
// either end (so the spine mouth is capped) or a t_cross in the middle.
func junctionName(dir byte, i, n int) string {
	left, right := i == 0, i == n-1
	if dir == 'U' {
		switch {
		case left:
			return "turn_left_bottom"
		case right:
			return "turn_right_bottom"
		default:
			return "t_cross_up"
		}
	}
	switch {
	case left:
		return "turn_left_upper"
	case right:
		return "turn_right_upper"
	default:
		return "t_cross_down"
	}
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
	ps.hFloorTop, ps.hChannel = firstRun(func(i int) bool { return open(h.tx+h.tw/2, h.ty+i) }, h.th)
	ps.vFloorLeft, ps.vChannel = firstRun(func(i int) bool { return open(v.tx+i, v.ty+v.th/2) }, v.tw)
	if ps.hChannel == 0 || ps.vChannel == 0 {
		return passageSet{}, fmt.Errorf("could not detect passage channels")
	}
	return ps, nil
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

// pickCorridorGids chooses the floor and wall tiles used by the corridors. Both
// are sampled from the passage pieces: the most common floor tile and the most
// common light-absorbing (collidable) wall tile within the passage blocks. The
// wall tile is used to seal unused room doors.
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
