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
// Layout is a ladder of horizontal "spine" corridors, not a grid. Rooms are
// split into rows; each row gets one spine, tiled from whole horizontal straight
// pieces, with its rooms branching off through their real entrances:
//
//   - a north door -> room sits below the spine, reached by a t_cross_down + a
//     vertical riser straight into the room's top;
//   - a south door -> room sits above the spine, reached by a t_cross_up + a riser
//     into the room's bottom;
//   - an east/west door -> a vertical riser drops off the spine and a turn piece
//     bends it horizontally into the room's side.
//
// The spines are joined at both ends by vertical "trunk" corridors (turn pieces
// at the top/bottom spine, t_cross pieces where a trunk passes a middle spine).
// The two trunks plus the spines form loops, so most rooms have more than one
// route between them. A single-row layout instead caps its spine ends with turns
// that reach the first/last room.
//
// Every door of every placed room is connected, never sealed: extra doors on the
// same side each get their own riser, and a door facing away from the room's spine
// (a south door on a below-the-spine room) is wired back to that spine with a
// U-shaped detour that runs under the room — another loop. Only a lone room (no
// network to join) has its doors closed with the corridor wall tile. Every piece
// carries its own walls, so corridors are self-sealing and there is no
// fill/auto-wall step; piece stamping skips void tiles so a piece's transparent
// corners never erase a neighbour.

const (
	genMargin    = 2  // void border around the whole map
	genGap       = 4  // horizontal gap between rooms along the spine
	genUpRiser   = 16 // clearance above the spine for south-door rooms
	genDownRiser = 9  // clearance below the spine before north-door rooms
	genStub      = 3  // short horizontal stub into an east/west room
	genTRight    = 18 // how far a spine junction reaches right of its channel
	genReach     = 10 // how far a connected door is carved through the room wall
	genGapRows   = 6  // vertical gap between one row's rooms and the next row
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

// slot is a placed room plus the geometry of the corridor branches that connect
// its doors to the spine.
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
	awayLaneCx int    // column of the return lane for a south door facing away (0 = none)
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

	// Classify every room by the door its corridor will use.
	slots := make([]slot, n)
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
		slots[i] = s
	}

	if n == 1 {
		return generateSingleRoom(template, &slots[0], wallGid)
	}

	// Split rooms into rows so the map folds into a compact rectangle instead of
	// one very long spine.
	perRow := max(1, int(math.Ceil(math.Sqrt(float64(n)))))
	var rows [][]int
	for i := 0; i < n; i += perRow {
		rows = append(rows, indexRange(i, min(i+perRow, n)))
	}
	R := len(rows)

	// Consecutive branch junctions must be at least a t_cross apart so their
	// pieces never overlap and sever a spine.
	minGap := pass.piece("t_cross_down").tw
	trunkLeftX := genMargin + 2

	// --- Horizontal layout: lay each row left to right, assigning every room an x
	// position and the column of its branch's vertical channel. Rooms start right
	// of the left trunk's widest junction so the trunk never cuts into one. ---
	startX := trunkLeftX + pass.piece("t_cross_right").tw + genGap
	maxRight := 0
	for _, ri := range rows {
		curX := startX
		prevCx := trunkLeftX
		for _, i := range ri {
			s := &slots[i]
			t := s.t
			switch {
			case !s.ew:
				s.roomX = curX
				s.cx = s.roomX + s.pe.off - (pass.vChannel-s.pe.openLen)/2
			case s.goWest: // east door: room first, riser to its right
				s.roomX = curX
				cornerX := s.roomX + t.tw + genStub
				s.cx = cornerX + pass.piece(s.cornerName).vChanOff
			default: // west door: riser first, room to its right
				s.cx = curX + genTRight
				corner := pass.piece(s.cornerName)
				s.roomX = s.cx - corner.vChanOff + corner.tw + genStub
			}
			// shift right so this room's first tap clears the previous room
			if d := prevCx + minGap - s.cx; d > 0 {
				s.roomX += d
				s.cx += d
			}
			// rightmost feature column: every door that taps this room's spine,
			// plus the return lane for a south door that faces away from it.
			rightTap := s.cx
			if !s.ew {
				for _, e := range t.entrances {
					if (s.dir == 'D' && e.side == sideNorth) || (s.dir == 'U' && e.side == sideSouth) {
						rightTap = max(rightTap, s.roomX+e.off-(pass.vChannel-e.openLen)/2)
					}
				}
				if s.dir == 'D' && hasSide(t, sideSouth) {
					s.awayLaneCx = rightTap + minGap
					rightTap = s.awayLaneCx
				}
			}
			prevCx = rightTap
			rightExtent := max(s.roomX+t.tw, rightTap+genTRight)
			curX = rightExtent + genGap
			maxRight = max(maxRight, rightExtent)
		}
	}
	// The right trunk sits clear of the widest room edge and every branch tap.
	trunkRightX := maxRight + pass.piece("t_cross_left").vChanOff + genGap
	canvasW := trunkRightX + genTRight + genMargin

	// --- Vertical layout: stack the rows, each with a band of rooms above and
	// below its spine. ---
	tDown := pass.piece("t_cross_down")
	tDownBelow := tDown.th - tDown.hChanOff // rows a t_cross_down reaches below the spine
	detourDepth := pass.piece("turn_right_bottom").th
	spineY := make([]int, R)
	belowTop := make([]int, R)
	y := genMargin
	for k, ri := range rows {
		aboveH, belowH := 0, 0
		for _, i := range ri {
			s := slots[i]
			if s.dir == 'U' {
				aboveH = max(aboveH, s.t.th)
			} else {
				h := s.t.th
				if s.ew {
					h = max(h, 28)
				}
				if s.awayLaneCx != 0 {
					h += detourDepth // the south-door detour runs below the room
				}
				belowH = max(belowH, h)
			}
		}
		spineY[k] = y + aboveH + genUpRiser
		belowTop[k] = spineY[k] + tDownBelow + genDownRiser
		y = belowTop[k] + belowH + genGapRows
	}
	canvasH := y - genGapRows + genMargin

	for k, ri := range rows {
		for _, i := range ri {
			s := &slots[i]
			if s.dir == 'U' {
				s.roomY = spineY[k] - genUpRiser - s.t.th
			} else {
				s.roomY = belowTop[k]
				if s.ew {
					s.ry = s.roomY + s.pe.off - (pass.hChannel-s.pe.openLen)/2
				}
			}
		}
	}

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
	// carveThroat opens a connected door inward through the room's wall ring by
	// clearing only collidable wall tiles (the floor already exists underneath).
	// Decorative tiles - door arches, jambs, pillars - are left in place.
	absorb := lightAbsorbingGids(template)
	carveThroat := func(rx, ry int, t roomTemplate, e entrance) {
		clear := func(x, y int) {
			if x >= 0 && x < canvasW && y >= 0 && y < canvasH {
				i := x + y*canvasW
				if !absorb[walls[i]] {
					return
				}
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
	// renderAway connects a south door that faces away from the room's spine with a
	// U-shaped detour: a riser drops off the spine in a return lane to the right,
	// turns under the room and rises back up into the door. This forms a loop.
	renderAway := func(s *slot, sy int) {
		var sDoor entrance
		for _, e := range s.t.entrances {
			if e.side == sideSouth {
				sDoor = e
				break
			}
		}
		sCx := s.roomX + sDoor.off - (pass.vChannel-sDoor.openLen)/2
		roomBottom := s.roomY + s.t.th
		c1 := pass.piece("turn_right_bottom") // lane corner: down -> west
		c2 := pass.piece("turn_left_bottom")  // door corner: east -> up
		uy := roomBottom + c1.hChanOff        // under-room channel row
		c1x, c1y := s.awayLaneCx-c1.vChanOff, uy-c1.hChanOff
		c2x, c2y := sCx-c2.vChanOff, uy-c2.hChanOff
		tp := stampJunction("t_cross_down", s.awayLaneCx, sy)
		tileV(s.awayLaneCx, sy-tp.hChanOff+tp.th, c1y-1)
		stamp("turn_right_bottom", c1x, c1y)
		stamp("turn_left_bottom", c2x, c2y)
		tileH(uy, c2x+c2.tw, c1x-1)
		tileV(sCx, roomBottom, c2y-1)
	}

	// renderRoomCorridors connects every door of one room to its spine. North/south
	// doors that face the spine get a straight riser; an east/west door bends in via
	// a turn; a south door facing away takes the return-lane detour.
	renderRoomCorridors := func(s *slot, sy int, primaryJname string) {
		switch {
		case s.ew:
			jp := stampJunction(primaryJname, s.cx, sy)
			corner := pass.piece(s.cornerName)
			cornerX := s.cx - corner.vChanOff
			cornerY := s.ry - corner.hChanOff
			tileV(s.cx, sy-jp.hChanOff+jp.th, cornerY-1)
			stamp(s.cornerName, cornerX, cornerY)
			if s.goWest {
				tileH(s.ry, s.roomX+s.t.tw, cornerX-1)
			} else {
				tileH(s.ry, cornerX+corner.tw, s.roomX-1)
			}
		case s.dir == 'U':
			jp := stampJunction(primaryJname, s.cx, sy)
			tileV(s.cx, s.roomY+s.t.th, sy-jp.hChanOff-1)
		default: // below the spine: every north door rises into it
			for _, e := range s.t.entrances {
				if e.side != sideNorth {
					continue
				}
				col := s.roomX + e.off - (pass.vChannel-e.openLen)/2
				jname := "t_cross_down"
				if e == s.pe {
					jname = primaryJname
				}
				jp := stampJunction(jname, col, sy)
				tileV(col, sy-jp.hChanOff+jp.th, s.roomY-1)
			}
			if s.awayLaneCx != 0 {
				renderAway(s, sy)
			}
		}
	}
	// renderTrunk stamps the junction at each spine along a vertical trunk and
	// tiles straight pieces between them.
	renderTrunk := func(trunkX int, nameFn func(k, R int) string) {
		prevBottom := -1
		for k := 0; k < R; k++ {
			jp := stampJunction(nameFn(k, R), trunkX, spineY[k])
			top := spineY[k] - jp.hChanOff
			if prevBottom >= 0 {
				tileV(trunkX, prevBottom+1, top-1)
			}
			prevBottom = top + jp.th - 1
		}
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
		for _, e := range s.t.entrances {
			carveThroat(s.roomX, s.roomY, s.t, e)
		}
	}

	// --- Spines + room taps ---
	for k, ri := range rows {
		var leftEnd, rightEnd int
		if R == 1 {
			rl := len(ri)
			lc := pass.piece(junctionName(slots[ri[0]].dir, 0, rl))
			rc := pass.piece(junctionName(slots[ri[rl-1]].dir, rl-1, rl))
			leftEnd = slots[ri[0]].cx - lc.vChanOff + lc.tw
			rightEnd = slots[ri[rl-1]].cx - rc.vChanOff
		} else {
			lj := pass.piece(leftJunctionName(k, R))
			rj := pass.piece(rightJunctionName(k, R))
			leftEnd = trunkLeftX - lj.vChanOff + lj.tw
			rightEnd = trunkRightX - rj.vChanOff
		}
		tileH(spineY[k], leftEnd, rightEnd-1)
		for idx, i := range ri {
			s := &slots[i]
			var jname string
			switch {
			case R == 1:
				jname = junctionName(s.dir, idx, len(ri))
			case s.dir == 'U':
				jname = "t_cross_up"
			default:
				jname = "t_cross_down"
			}
			renderRoomCorridors(s, spineY[k], jname)
		}
	}

	// --- Trunks join the spines into a ladder (loops) ---
	if R >= 2 {
		renderTrunk(trunkLeftX, leftJunctionName)
		renderTrunk(trunkRightX, rightJunctionName)
	}

	out.setObjectLayer("objects", objects)
	out.setObjectLayer("spawns", spawns)

	return out, nil
}

// generateSingleRoom builds a map holding just one room (all doors sealed).
func generateSingleRoom(template *Map, s *slot, wallGid int) (*Map, error) {
	t := s.t
	out := newBlankMap(template, t.tw+2*genMargin, t.th+2*genMargin)
	walls := out.tileLayerData("walls")
	if walls == nil {
		return nil, fmt.Errorf("template map is missing a walls layer")
	}
	ox, oy := genMargin, genMargin
	stampRoom(out, template, t, ox, oy)
	objects := appendRoomObjects(nil, template, t, ox, oy, new(int))
	spawns := appendRoomSpawns(nil, template, t, ox, oy, new(int))
	out.roomCenters = [][2]int{{ox + t.tw/2, oy + t.th/2}}
	out.spawnX = (ox + t.tw/2) * out.TileWidth
	out.spawnY = (oy + t.th/2) * out.TileHeight
	if t.isDemon {
		out.demonRoomCount++
	}
	for _, e := range t.entrances {
		for k := 0; k < e.openLen; k++ {
			set := func(x, y int) { walls[x+y*out.Width] = wallGid }
			switch e.side {
			case sideNorth:
				set(ox+e.off+k, oy)
			case sideSouth:
				set(ox+e.off+k, oy+t.th-1)
			case sideWest:
				set(ox, oy+e.off+k)
			case sideEast:
				set(ox+t.tw-1, oy+e.off+k)
			}
		}
	}
	out.setObjectLayer("objects", objects)
	out.setObjectLayer("spawns", spawns)
	return out, nil
}

// indexRange returns the slice [lo, lo+1, ..., hi-1].
func indexRange(lo, hi int) []int {
	out := make([]int, 0, hi-lo)
	for i := lo; i < hi; i++ {
		out = append(out, i)
	}
	return out
}

// leftJunctionName / rightJunctionName name the piece where a vertical trunk meets
// spine k of R: a turn at the top and bottom spine, a t_cross in between.
func leftJunctionName(k, R int) string {
	switch {
	case k == 0:
		return "turn_left_upper"
	case k == R-1:
		return "turn_left_bottom"
	default:
		return "t_cross_right"
	}
}

func rightJunctionName(k, R int) string {
	switch {
	case k == 0:
		return "turn_right_upper"
	case k == R-1:
		return "turn_right_bottom"
	default:
		return "t_cross_left"
	}
}

// lightAbsorbingGids returns the set of tile gids flagged absorbs_light, i.e. the
// tiles that actually block movement and sight. Other wall-layer tiles (door
// arches, jambs, pillars) are decorative and passable.
func lightAbsorbingGids(m *Map) map[int]bool {
	absorbs := make(map[int]bool)
	for _, set := range m.Tilesets {
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
	return absorbs
}

// hasSide reports whether the room has at least one door on the given side.
func hasSide(t roomTemplate, side roomSide) bool {
	for _, e := range t.entrances {
		if e.side == side {
			return true
		}
	}
	return false
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
	absorbs := lightAbsorbingGids(template)
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
