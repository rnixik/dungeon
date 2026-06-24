package game

import "testing"

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

func TestGenerateMapConnectivity(t *testing.T) {
	template, err := parseMap("../../public/assets/dungeon1.tmj")
	if err != nil {
		t.Fatalf("parse template: %v", err)
	}

	for _, n := range []int{1, 2, 4, 6, 9, 10, 13, 20} {
		for seed := int64(1); seed <= 6; seed++ {
			m, err := generateMap(template, n, seed)
			if err != nil {
				t.Fatalf("generate (n=%d seed=%d): %v", n, seed, err)
			}
			sx, sy := m.PlayerSpawn()
			reach := reachableFloor(m, sx/m.TileWidth, sy/m.TileHeight)
			if len(reach) == 0 {
				t.Fatalf("n=%d seed=%d: spawn (%d,%d)px not on open floor", n, seed, sx, sy)
			}
			for _, c := range m.roomCenters {
				if !reach[c[0]+c[1]*m.Width] {
					t.Errorf("n=%d seed=%d: room interior (%d,%d) unreachable", n, seed, c[0], c[1])
				}
			}
			if m.demonRoomCount > 1 {
				t.Errorf("n=%d seed=%d: demon room placed %d times", n, seed, m.demonRoomCount)
			}
		}
	}
}
