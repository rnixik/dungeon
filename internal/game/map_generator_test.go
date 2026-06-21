package game

import (
	"math/rand"
	"testing"
)

func newSeededRng(seed int64) *rand.Rand { return rand.New(rand.NewSource(seed)) }

// reachableFloor returns the open floor tiles reachable from the start tile.
func reachableFloor(m *Map, startX, startY int) map[int]bool {
	floor := m.tileLayerData("floor")
	walls := m.tileLayerData("walls")
	w, h := m.Width, m.Height
	seen := make(map[int]bool)
	idx := func(x, y int) int { return x + y*w }
	open := func(x, y int) bool {
		if x < 0 || x >= w || y < 0 || y >= h {
			return false
		}
		i := idx(x, y)
		return floor[i] != 0 && walls[i] == 0
	}
	if !open(startX, startY) {
		return seen
	}
	stack := []int{idx(startX, startY)}
	seen[stack[0]] = true
	for len(stack) > 0 {
		cur := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		cx, cy := cur%w, cur/w
		for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
			nx, ny := cx+d[0], cy+d[1]
			if open(nx, ny) && !seen[idx(nx, ny)] {
				seen[idx(nx, ny)] = true
				stack = append(stack, idx(nx, ny))
			}
		}
	}
	return seen
}

func mustTemplates(t *testing.T, template *Map) []roomTemplate {
	tmpls, err := extractRoomTemplates(template)
	if err != nil {
		t.Fatalf("extract templates: %v", err)
	}
	return tmpls
}

// placedRoomCenters recomputes where each placed room interior centre lands,
// mirroring generateMap's layout math.
func placedRoomCenters(templates []roomTemplate, pass passageSet, n int, rng *rand.Rand) (centers [][2]int, demonCount int) {
	place := chooseRooms(templates, n, rng)
	cols, rows := gridDims(n)
	cell := assignCells(place, cols, rows)
	gapX, gapY := pass.cr.tw, pass.cr.th
	colWidth := make([]int, cols)
	rowHeight := make([]int, rows)
	for r := 0; r < rows; r++ {
		for c := 0; c < cols; c++ {
			if pi := cell[r*cols+c]; pi >= 0 {
				if place[pi].tw > colWidth[c] {
					colWidth[c] = place[pi].tw
				}
				if place[pi].th > rowHeight[r] {
					rowHeight[r] = place[pi].th
				}
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
	for _, t := range place {
		if t.isDemon {
			demonCount++
		}
	}
	for r := 0; r < rows; r++ {
		for c := 0; c < cols; c++ {
			pi := cell[r*cols+c]
			if pi < 0 {
				continue
			}
			t := place[pi]
			centers = append(centers, [2]int{roomX[c] + t.tw/2, roomY[r] + t.th/2})
		}
	}
	return centers, demonCount
}

func TestGenerateMapConnectivity(t *testing.T) {
	template, err := parseMap("../../public/assets/dungeon1.tmj")
	if err != nil {
		t.Fatalf("parse template: %v", err)
	}
	tmpls := mustTemplates(t, template)
	pass, err := extractPassages(template)
	if err != nil {
		t.Fatalf("extract passages: %v", err)
	}

	for _, n := range []int{1, 2, 4, 6, 9, 10, 13, 20} {
		for seed := int64(1); seed <= 6; seed++ {
			m, err := generateMap(template, n, seed)
			if err != nil {
				t.Fatalf("generate (n=%d seed=%d): %v", n, seed, err)
			}
			spawnTX, spawnTY := 120/m.TileWidth, 140/m.TileHeight
			reach := reachableFloor(m, spawnTX, spawnTY)
			if len(reach) == 0 {
				t.Fatalf("n=%d seed=%d: spawn (%d,%d) not on open floor", n, seed, spawnTX, spawnTY)
			}
			centers, demonCount := placedRoomCenters(tmpls, pass, n, newSeededRng(seed))
			for _, c := range centers {
				if !reach[c[0]+c[1]*m.Width] {
					t.Errorf("n=%d seed=%d: room interior (%d,%d) unreachable", n, seed, c[0], c[1])
				}
			}
			if demonCount > 1 {
				t.Errorf("n=%d seed=%d: demon room placed %d times", n, seed, demonCount)
			}
		}
	}
}
