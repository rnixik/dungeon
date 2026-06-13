package game

import "testing"

func TestAbs(t *testing.T) {
	cases := map[int]int{0: 0, 5: 5, -5: 5, -1: 1}
	for in, want := range cases {
		if got := abs(in); got != want {
			t.Errorf("abs(%d) = %d, want %d", in, got, want)
		}
	}
}

func TestGetVectorFromDirection(t *testing.T) {
	tests := []struct {
		dir   string
		wantX float64
		wantY float64
	}{
		{"up", 0, -1},
		{"down", 0, 1},
		{"left", -1, 0},
		{"right", 1, 0},
		{"", 0, 0},
		{"diagonal", 0, 0},
	}
	for _, tc := range tests {
		x, y := getVectorFromDirection(tc.dir)
		if x != tc.wantX || y != tc.wantY {
			t.Errorf("getVectorFromDirection(%q) = (%v, %v), want (%v, %v)", tc.dir, x, y, tc.wantX, tc.wantY)
		}
	}
}

func TestGetDirection(t *testing.T) {
	tests := []struct {
		name           string
		x1, y1, x2, y2 int
		want           string
	}{
		{"east", 0, 0, 10, 0, "right"},
		{"west", 0, 0, -10, 0, "left"},
		{"south", 0, 0, 0, 10, "down"},
		{"north", 0, 0, 0, -10, "up"},
		{"dominant horizontal", 0, 0, 10, 3, "right"},
		{"dominant vertical", 0, 0, 3, 10, "down"},
		{"tie favors vertical", 0, 0, 5, 5, "down"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := getDirection(tc.x1, tc.y1, tc.x2, tc.y2); got != tc.want {
				t.Errorf("getDirection = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestGetDistance(t *testing.T) {
	// Manhattan distance.
	if got := getDistance(0, 0, 3, 4); got != 7 {
		t.Errorf("getDistance = %d, want 7", got)
	}
	if got := getDistance(5, 5, 5, 5); got != 0 {
		t.Errorf("getDistance same point = %d, want 0", got)
	}
	if got := getDistance(2, 2, -1, -2); got != 7 {
		t.Errorf("getDistance with negatives = %d, want 7", got)
	}
}

func TestPointInRect(t *testing.T) {
	// rect at (0,0) size 10x10.
	tests := []struct {
		name   string
		px, py int
		want   bool
	}{
		{"center", 5, 5, true},
		{"top-left corner", 0, 0, true},
		{"bottom-right corner", 10, 10, true},
		{"outside right", 11, 5, false},
		{"outside above", 5, -1, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := pointInRect(tc.px, tc.py, 0, 0, 10, 10); got != tc.want {
				t.Errorf("pointInRect(%d,%d) = %v, want %v", tc.px, tc.py, got, tc.want)
			}
		})
	}
}

func TestLinesIntersect(t *testing.T) {
	tests := []struct {
		name                           string
		x1, y1, x2, y2, x3, y3, x4, y4 int
		want                           bool
	}{
		{"crossing X", 0, 0, 10, 10, 0, 10, 10, 0, true},
		{"parallel no touch", 0, 0, 10, 0, 0, 5, 10, 5, false},
		{"disjoint", 0, 0, 1, 1, 5, 5, 6, 6, false},
		{"touch at endpoint", 0, 0, 5, 5, 5, 5, 10, 0, true},
		{"collinear overlap", 0, 0, 10, 0, 5, 0, 15, 0, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := linesIntersect(tc.x1, tc.y1, tc.x2, tc.y2, tc.x3, tc.y3, tc.x4, tc.y4)
			if got != tc.want {
				t.Errorf("linesIntersect = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestLineIntersectsRect(t *testing.T) {
	// rect at (10,10) size 10x10 -> spans (10,10)..(20,20).
	const rx, ry, rw, rh = 10, 10, 10, 10
	tests := []struct {
		name           string
		x1, y1, x2, y2 int
		want           bool
	}{
		{"line crosses rect", 0, 15, 30, 15, true},
		{"endpoint inside rect", 15, 15, 30, 30, true},
		{"line misses rect", 0, 0, 5, 5, false},
		{"line passes above", 0, 0, 30, 0, false},
		{"line clips edge", 10, 0, 10, 30, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := lineIntersectsRect(tc.x1, tc.y1, tc.x2, tc.y2, rx, ry, rw, rh)
			if got != tc.want {
				t.Errorf("lineIntersectsRect = %v, want %v", got, tc.want)
			}
		})
	}
}
