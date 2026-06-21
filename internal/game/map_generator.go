package game

import (
	"fmt"
	"math"
	"math/rand"
	"sort"
	"time"
)

// Map generation assembles a fresh map from the predefined templates in a
// template .tmj file:
//
//   - Rooms (class="room") are blocks with thick walls and one or more
//     class="entrance" openings (matched by name). Two rooms are special:
//     "room_start" (where players spawn) and the room containing the demon boss
//     spawn ("room_boss").
//   - Passages (class="passage") are whole corridor tile pieces: "horizontal"
//     and "vertical" straights, four "turn_*" L-bends, and a 4-way "crossroad".
//
// Rooms are laid out on a grid with variable column widths and row heights (so
// large rooms do not bloat every cell). Between adjacent room columns runs a
// full-height vertical street and between adjacent rows a full-width horizontal
// street; they cross at crossroads. Streets are rendered by stamping the whole
// straight passage pieces end to end (never copying single tiles), and the
// crossroad piece is stamped at each intersection. Every room is connected to
// the street grid through its existing entrances only. A final pass turns void
// tiles touching floor into walls.

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
	off     int // tile offset of the opening along its edge
	openLen int
}

// roomTemplate describes one predefined room. Coordinates are tiles relative to
// the template map.
type roomTemplate struct {
	name           string
	tx, ty, tw, th int
	entrances      []entrance
	isStart        bool
	isDemon        bool
}

// passageSet holds the corridor pieces and their detected channel geometry.
type passageSet struct {
	h  passagePiece // horizontal straight
	v  passagePiece // vertical straight
	cr passagePiece // crossroad

	hFloorTop  int // first open row of the horizontal channel in the straight
	hChannel   int // horizontal channel height
	vFloorLeft int // first open col of the vertical channel in the straight
	vChannel   int // vertical channel width
	cVOff      int // first open col of the crossroad's vertical channel
	cHOff      int // first open row of the crossroad's horizontal channel
}

type passagePiece struct {
	tx, ty, tw, th int
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
	floorGid, wallGid, err := pickCorridorGids(template, templates)
	if err != nil {
		return nil, err
	}

	place := chooseRooms(templates, numRooms, rng)
	cols, rows := gridDims(numRooms)
	cell := assignCells(place, cols, rows) // cell[r*cols+c] = index into place, or -1

	// Variable column widths and row heights from the assigned rooms.
	gapX, gapY := pass.cr.tw, pass.cr.th
	colWidth := make([]int, cols)
	rowHeight := make([]int, rows)
	for r := 0; r < rows; r++ {
		for c := 0; c < cols; c++ {
			if pi := cell[r*cols+c]; pi >= 0 {
				colWidth[c] = max(colWidth[c], place[pi].tw)
				rowHeight[r] = max(rowHeight[r], place[pi].th)
			}
		}
	}

	roomX := make([]int, cols)
	roomY := make([]int, rows)
	roomX[0] = genMargin
	for c := 1; c < cols; c++ {
		roomX[c] = roomX[c-1] + colWidth[c-1] + gapX
	}
	roomY[0] = genMargin
	for r := 1; r < rows; r++ {
		roomY[r] = roomY[r-1] + rowHeight[r-1] + gapY
	}
	canvasW := roomX[cols-1] + colWidth[cols-1] + genMargin
	canvasH := roomY[rows-1] + rowHeight[rows-1] + genMargin

	// Street channel positions.
	vGapStart := func(c int) int { return roomX[c] + colWidth[c] }    // left of the gap after column c
	hGapStart := func(r int) int { return roomY[r] + rowHeight[r] }   // top of the gap after row r
	vChanLeft := func(c int) int { return vGapStart(c) + pass.cVOff } // vertical street channel left col
	hChanTop := func(r int) int { return hGapStart(r) + pass.cHOff }  // horizontal street channel top row
	gridTop, gridBottom := roomY[0], roomY[rows-1]+rowHeight[rows-1]
	gridLeft, gridRight := roomX[0], roomX[cols-1]+colWidth[cols-1]

	out := newBlankMap(template, canvasW, canvasH)
	floor := out.tileLayerData("floor")
	walls := out.tileLayerData("walls")
	if floor == nil || walls == nil {
		return nil, fmt.Errorf("template map is missing a floor or walls layer")
	}
	carve := func(x0, x1, y0, y1 int) {
		if x0 > x1 {
			x0, x1 = x1, x0
		}
		if y0 > y1 {
			y0, y1 = y1, y0
		}
		for y := max(0, y0); y <= min(canvasH-1, y1); y++ {
			for x := max(0, x0); x <= min(canvasW-1, x1); x++ {
				floor[x+y*canvasW] = floorGid
				walls[x+y*canvasW] = 0
			}
		}
	}

	objects := make([]MapObject, 0)
	spawns := make([]MapObject, 0)
	nextID := 1

	// --- Stamp rooms ---
	for r := 0; r < rows; r++ {
		for c := 0; c < cols; c++ {
			pi := cell[r*cols+c]
			if pi < 0 {
				continue
			}
			t := place[pi]
			ox, oy := roomX[c], roomY[r]
			stampRoom(out, template, t, ox, oy)
			objects = appendRoomObjects(objects, template, t, ox, oy, &nextID)
			spawns = appendRoomSpawns(spawns, template, t, ox, oy, &nextID)
		}
	}

	// --- Streets from whole passage pieces ---
	for c := 0; c < cols-1; c++ {
		extrudeVertical(out, template, pass, vChanLeft(c), gridTop, gridBottom)
	}
	for r := 0; r < rows-1; r++ {
		extrudeHorizontal(out, template, pass, hChanTop(r), gridLeft, gridRight)
	}
	for r := 0; r < rows-1; r++ {
		for c := 0; c < cols-1; c++ {
			copyBlock(out, template, pass.cr.tx, pass.cr.ty, pass.cr.tw, pass.cr.th, vGapStart(c), hGapStart(r))
		}
	}

	// --- Connect every room to the streets through its existing entrances ---
	hasStreet := func(r, c int, s roomSide) bool {
		switch s {
		case sideNorth:
			return r > 0
		case sideSouth:
			return r < rows-1
		case sideWest:
			return c > 0
		case sideEast:
			return c < cols-1
		}
		return false
	}
	for r := 0; r < rows; r++ {
		for c := 0; c < cols; c++ {
			pi := cell[r*cols+c]
			if pi < 0 {
				continue
			}
			t := place[pi]
			ox, oy := roomX[c], roomY[r]
			for _, e := range t.entrances {
				if hasStreet(r, c, e.side) {
					connectStraight(carve, pass, t, ox, oy, e, r, c, vChanLeft, hChanTop)
				} else {
					connectElbow(carve, pass, t, ox, oy, e, r, c, cols, rows, vChanLeft, hChanTop)
				}
			}
		}
	}

	autoWall(floor, walls, canvasW, canvasH, wallGid)

	out.setObjectLayer("objects", objects)
	out.setObjectLayer("spawns", spawns)

	return out, nil
}

// connectStraight carves a corridor from one entrance straight to the street on
// that entrance's side.
func connectStraight(carve func(x0, x1, y0, y1 int), p passageSet, t roomTemplate, ox, oy int, e entrance, r, c int, vChanLeft, hChanTop func(int) int) {
	a := e.off
	b := e.off + e.openLen - 1
	switch e.side {
	case sideSouth:
		carve(ox+a, ox+b, oy+t.th-genReach, hChanTop(r)+p.hChannel-1)
	case sideNorth:
		carve(ox+a, ox+b, hChanTop(r-1), oy+genReach)
	case sideEast:
		carve(ox+t.tw-genReach, vChanLeft(c)+p.vChannel-1, oy+a, oy+b)
	case sideWest:
		carve(vChanLeft(c-1), ox+genReach, oy+a, oy+b)
	}
}

// connectElbow handles an entrance whose own side has no street (an outward edge
// entrance): it pokes out of the entrance, then turns to a perpendicular street.
func connectElbow(carve func(x0, x1, y0, y1 int), p passageSet, t roomTemplate, ox, oy int, e entrance, r, c, cols, rows int, vChanLeft, hChanTop func(int) int) {
	a := e.off
	b := e.off + e.openLen - 1
	const out = 4
	switch e.side {
	case sideNorth, sideSouth:
		// Poke vertically into the margin, then run horizontally to E or W street.
		var bandY0, bandY1 int
		if e.side == sideNorth {
			bandY0, bandY1 = oy-out, oy+genReach
		} else {
			bandY0, bandY1 = oy+t.th-genReach, oy+t.th+out
		}
		carve(ox+a, ox+b, bandY0, bandY1)
		bandTop := min(bandY0, bandY1)
		if c < cols-1 { // east street
			carve(ox+a, vChanLeft(c)+p.vChannel-1, bandTop, bandTop+p.vChannel-1)
		} else if c > 0 { // west street
			carve(vChanLeft(c-1), ox+b, bandTop, bandTop+p.vChannel-1)
		}
	case sideEast, sideWest:
		var bandX0, bandX1 int
		if e.side == sideEast {
			bandX0, bandX1 = ox+t.tw-genReach, ox+t.tw+out
		} else {
			bandX0, bandX1 = ox-out, ox+genReach
		}
		carve(bandX0, bandX1, oy+a, oy+b)
		bandLeft := min(bandX0, bandX1)
		if r < rows-1 { // south street
			carve(bandLeft, bandLeft+p.hChannel-1, oy+a, hChanTop(r)+p.hChannel-1)
		} else if r > 0 { // north street
			carve(bandLeft, bandLeft+p.hChannel-1, hChanTop(r-1), oy+b)
		}
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
	if start == nil { // fall back to any room as the spawn room
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

// assignCells maps the rooms in place onto the first len(place) grid cells so
// that each room has at least one entrance facing an interior street where
// possible. cell[0] is always room_start; the demon room is kept off the top
// row. Returns cell -> place index (or -1 for empty cells).
func assignCells(place []roomTemplate, cols, rows int) []int {
	n := len(place)
	cell := make([]int, cols*rows)
	for i := range cell {
		cell[i] = -1
	}
	streetSides := func(idx int) map[roomSide]bool {
		r, c := idx/cols, idx%cols
		s := map[roomSide]bool{}
		if r > 0 {
			s[sideNorth] = true
		}
		if r < rows-1 {
			s[sideSouth] = true
		}
		if c > 0 {
			s[sideWest] = true
		}
		if c < cols-1 {
			s[sideEast] = true
		}
		return s
	}
	fits := func(t roomTemplate, idx int) bool {
		sides := streetSides(idx)
		for _, e := range t.entrances {
			if sides[e.side] {
				return true
			}
		}
		return false
	}

	free := make([]bool, cols*rows)
	for i := 0; i < n; i++ {
		free[i] = true
	}
	take := func(pi, idx int) {
		cell[idx] = pi
		free[idx] = false
	}

	// room_start at cell 0 (top-left, contains the player spawn).
	startPi := -1
	for i := range place {
		if place[i].isStart {
			startPi = i
			break
		}
	}
	if startPi < 0 {
		startPi = 0
	}
	take(startPi, 0)

	// demon room on a fitting, non-top-row cell (prefer the last cell).
	demonPi := -1
	for i := range place {
		if place[i].isDemon {
			demonPi = i
			break
		}
	}
	if demonPi >= 0 {
		placed := false
		for idx := cols * rows; idx > 0; idx-- {
			i := idx - 1
			if i < n && free[i] && i/cols > 0 && fits(place[demonPi], i) {
				take(demonPi, i)
				placed = true
				break
			}
		}
		if !placed {
			for i := 0; i < n; i++ {
				if free[i] {
					take(demonPi, i)
					break
				}
			}
		}
	}

	// Remaining rooms, most-constrained first.
	type rem struct{ pi, opts int }
	var rems []rem
	for i := range place {
		if i == startPi || i == demonPi {
			continue
		}
		opts := 0
		for idx := 0; idx < n; idx++ {
			if free[idx] && fits(place[i], idx) {
				opts++
			}
		}
		rems = append(rems, rem{i, opts})
	}
	sort.Slice(rems, func(a, b int) bool { return rems[a].opts < rems[b].opts })
	for _, rm := range rems {
		best := -1
		for idx := 0; idx < n; idx++ {
			if free[idx] && fits(place[rm.pi], idx) {
				best = idx
				break
			}
		}
		if best < 0 { // no fitting cell; take any free one (elbow connectors cope)
			for idx := 0; idx < n; idx++ {
				if free[idx] {
					best = idx
					break
				}
			}
		}
		if best >= 0 {
			take(rm.pi, best)
		}
	}
	return cell
}

// extractRoomTemplates reads every class="room" object, its class="entrance"
// openings, and whether it is the start room or contains the demon spawn.
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

// extractPassages reads the corridor pieces and detects channel geometry.
func extractPassages(template *Map) (passageSet, error) {
	objectsLayer := template.objectLayer("objects")
	if objectsLayer == nil {
		return passageSet{}, fmt.Errorf("template map has no objects layer")
	}
	ts := template.TileWidth
	pieces := make(map[string]passagePiece)
	for _, o := range objectsLayer.Objects {
		if o.Type == "passage" {
			pieces[o.Name] = passagePiece{int(o.X) / ts, int(o.Y) / ts, int(o.Width) / ts, int(o.Height) / ts}
		}
	}
	need := []string{"horizontal", "vertical", "crossroad"}
	for _, name := range need {
		if _, ok := pieces[name]; !ok {
			return passageSet{}, fmt.Errorf("template map is missing passage %q", name)
		}
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

	h, v, cr := pieces["horizontal"], pieces["vertical"], pieces["crossroad"]
	ps := passageSet{h: h, v: v, cr: cr}
	ps.hFloorTop, ps.hChannel = firstRun(func(i int) bool { return open(h.tx+h.tw/2, h.ty+i) }, h.th)
	ps.vFloorLeft, ps.vChannel = firstRun(func(i int) bool { return open(v.tx+i, v.ty+v.th/2) }, v.tw)
	ps.cVOff, _ = firstRun(func(i int) bool { return open(cr.tx+i, cr.ty) }, cr.tw)
	ps.cHOff, _ = firstRun(func(i int) bool { return open(cr.tx, cr.ty+i) }, cr.th)
	if ps.hChannel == 0 || ps.vChannel == 0 {
		return passageSet{}, fmt.Errorf("could not detect passage channels")
	}
	return ps, nil
}

// extrudeHorizontal stamps whole horizontal straight pieces end to end along a
// run, aligning the channel to rows [channelTop, channelTop+hChannel).
func extrudeHorizontal(out, template *Map, p passageSet, channelTop, x0, x1 int) {
	dstTop := channelTop - p.hFloorTop
	for x := x0; x <= x1; x += p.h.tw {
		copyBlock(out, template, p.h.tx, p.h.ty, min(p.h.tw, x1-x+1), p.h.th, x, dstTop)
	}
}

// extrudeVertical stamps whole vertical straight pieces end to end along a run,
// aligning the channel to cols [channelLeft, channelLeft+vChannel).
func extrudeVertical(out, template *Map, p passageSet, channelLeft, y0, y1 int) {
	dstLeft := channelLeft - p.vFloorLeft
	for y := y0; y <= y1; y += p.v.th {
		copyBlock(out, template, p.v.tx, p.v.ty, p.v.tw, min(p.v.th, y1-y+1), dstLeft, y)
	}
}

// copyBlock copies a w x h tile block from the template into the output map at
// the given tile offset, across every tile layer.
func copyBlock(out, template *Map, srcX, srcY, w, h, dstX, dstY int) {
	for li := range out.Layers {
		dst := &out.Layers[li]
		if dst.Type != "tilelayer" {
			continue
		}
		src := template.tileLayerData(dst.Name)
		if src == nil {
			continue
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
				dst.Data[nx+ny*out.Width] = src[sx+sy*template.Width]
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

// pickCorridorGids chooses the floor tile for connector corridors (sampled from a
// room interior) and the wall tile to seal them (the most common light-absorbing
// wall tile).
func pickCorridorGids(template *Map, templates []roomTemplate) (floorGid, wallGid int, err error) {
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
	t := templates[0]
	floorGid = floorLayer[(t.tx+t.tw/2)+(t.ty+t.th/2)*template.Width]
	if floorGid == 0 {
		return 0, 0, fmt.Errorf("could not sample a floor tile from room %q", t.name)
	}
	counts := make(map[int]int)
	for _, g := range wallLayer {
		if absorbs[g] {
			counts[g]++
		}
	}
	best := 0
	for g, ct := range counts {
		if ct > best {
			best, wallGid = ct, g
		}
	}
	if best == 0 {
		return 0, 0, fmt.Errorf("template walls layer has no light-absorbing tiles")
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
