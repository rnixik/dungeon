package game

import (
	"encoding/json"
	"fmt"
	"os"
)

type MapObject struct {
	Id         int           `json:"id"`
	Name       string        `json:"name"`
	Type       string        `json:"type"`
	Rotation   float64       `json:"rotation"`
	Visible    bool          `json:"visible"`
	Width      float64       `json:"width"`
	Height     float64       `json:"height"`
	X          float64       `json:"x"`
	Y          float64       `json:"y"`
	Properties []MapProperty `json:"properties,omitempty"`
	Point      bool          `json:"point,omitempty"`
}

type MapLayer struct {
	Data      []int       `json:"data,omitempty"`
	Objects   []MapObject `json:"objects"`
	Name      string      `json:"name"`
	Width     int         `json:"width,omitempty"`
	Height    int         `json:"height,omitempty"`
	Type      string      `json:"type"`
	X         int         `json:"x"`
	Y         int         `json:"y"`
	Visible   bool        `json:"visible"`
	ID        int         `json:"id"`
	Draworder string      `json:"draworder,omitempty"`
	Opacity   float64     `json:"opacity,omitempty"`
}

type MapTileset struct {
	Columns     int       `json:"columns"`
	Firstgid    int       `json:"firstgid"`
	Image       string    `json:"image"`
	Imageheight int       `json:"imageheight"`
	Imagewidth  int       `json:"imagewidth"`
	Margin      int       `json:"margin"`
	Name        string    `json:"name"`
	Spacing     int       `json:"spacing"`
	Tilecount   int       `json:"tilecount"`
	Tileheight  int       `json:"tileheight"`
	Tilewidth   int       `json:"tilewidth"`
	Tiles       []MapTile `json:"tiles,omitempty"`
}

type MapProperty struct {
	Name  string `json:"name"`
	Type  string `json:"type"`
	Value any    `json:"value"`
}

type MapTile struct {
	Id         int           `json:"id"`
	Properties []MapProperty `json:"properties"`
}

type Point struct {
	X, Y int
}

type Map struct {
	Compression         int                    `json:"compressionlevel"`
	Infinite            bool                   `json:"infinite"`
	Width               int                    `json:"width"`
	Height              int                    `json:"height"`
	Layers              []MapLayer             `json:"layers"`
	TileWidth           int                    `json:"tilewidth"`
	TileHeight          int                    `json:"tileheight"`
	Tilesets            []MapTileset           `json:"tilesets"`
	Orientation         string                 `json:"orientation"`
	RenderOrder         string                 `json:"renderorder"`
	Type                string                 `json:"type"`
	Version             string                 `json:"version"`
	TilesPropertiesHash map[int]map[string]any `json:"-"`
	blockedGrid         []bool
	gridWidth           int
	gridHeight          int
	visibilityColliders []Rectangle
}

func LoadMap(filename string) (*Map, error) {
	var m Map
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(content, &m)
	if err != nil {
		return nil, err
	}

	m.fillTilesPropertiesHash()

	err = m.addLayerWithCollisionRectangles()
	if err != nil {
		return nil, err
	}

	m.buildAreaOptimizedCollisionRects()
	m.buildBlockedGrid()
	m.buildVisibilityColliders()

	return &m, err
}

func (m *Map) fillTilesPropertiesHash() {
	m.TilesPropertiesHash = make(map[int]map[string]any)
	for _, ts := range m.Tilesets {
		for _, tile := range ts.Tiles {
			gid := ts.Firstgid + tile.Id
			m.TilesPropertiesHash[gid] = make(map[string]any, len(tile.Properties))
			for _, p := range tile.Properties {
				m.TilesPropertiesHash[gid][p.Name] = p.Value
			}
		}
	}
}

func (m *Map) getLayerByName(layerName string) (layer *MapLayer) {
	for _, l := range m.Layers {
		if l.Name == layerName {
			layer = &l
			break
		}
	}

	return
}

type Rectangle struct {
	X      int
	Y      int
	Width  int
	Height int
}

func (m *Map) addLayerWithCollisionRectangles() error {
	wallLayer := m.getLayerByName("walls")
	if wallLayer == nil {
		return fmt.Errorf("no wall layer found in map")
	}

	rects := make([]Rectangle, 0)

	mapW := wallLayer.Width
	mapH := wallLayer.Height
	tw := m.TileWidth
	th := m.TileHeight

	isSolidAt := func(x, y int) bool {
		if x < 0 || x >= mapW || y < 0 || y >= mapH {
			return false
		}
		tileIndex := x + y*mapW
		if tileIndex < 0 || tileIndex >= len(wallLayer.Data) {
			return false
		}
		t := wallLayer.Data[tileIndex]
		if t == 0 {
			return false
		}

		if props, ok := m.TilesPropertiesHash[t]; ok {
			if collides, ok2 := props["absorbs_light"].(bool); ok2 {
				return collides
			}
		}

		return false
	}

	// 1) horizontal runs per row
	runs := make([][]struct{ x, w int }, mapH)
	for y := 0; y < mapH; y++ {
		x := 0
		for x < mapW {
			if !isSolidAt(x, y) {
				x++
				continue
			}
			x0 := x
			for x < mapW && isSolidAt(x, y) {
				x++
			}
			runs[y] = append(runs[y], struct{ x, w int }{x: x0, w: x - x0})
		}
	}

	// 2) vertical merge of identical runs
	used := make([][]bool, mapH)
	for y := range runs {
		used[y] = make([]bool, len(runs[y]))
	}

	for y := 0; y < mapH; y++ {
		for i := 0; i < len(runs[y]); i++ {
			if used[y][i] {
				continue
			}
			rx := runs[y][i].x
			rw := runs[y][i].w
			h := 1
			// try to extend downwards while the exact same run exists and not used
			yy := y + 1
			for yy < mapH {
				foundIdx := -1
				for j := 0; j < len(runs[yy]); j++ {
					if !used[yy][j] && runs[yy][j].x == rx && runs[yy][j].w == rw {
						foundIdx = j
						break
					}
				}
				if foundIdx == -1 {
					break
				}
				used[yy][foundIdx] = true
				h++
				yy++
			}
			used[y][i] = true
			rects = append(rects, Rectangle{X: rx * tw, Y: y * th, Width: rw * tw, Height: h * th})
		}
	}

	collisionObjects := make([]MapObject, 0, len(rects))

	for i, r := range rects {
		collisionObjects = append(collisionObjects, MapObject{
			Id:       i + 1,
			Name:     "collision",
			Type:     "collision",
			Rotation: 0,
			Visible:  true,
			Width:    float64(r.Width),
			Height:   float64(r.Height),
			X:        float64(r.X),
			Y:        float64(r.Y),
			Properties: []MapProperty{
				{
					Name:  "collides",
					Type:  "bool",
					Value: true,
				},
			},
		})
	}

	m.Layers = append(m.Layers, MapLayer{
		Data:    nil,
		Objects: collisionObjects,
		Name:    "collision-rects",
		Width:   0,
		Height:  0,
		Type:    "objectgroup",
		X:       0,
		Y:       0,
	})

	return nil
}

func (m *Map) buildAreaOptimizedCollisionRects() {
	// Split the map into overlapping areas of visibility
	w := 960
	h := 720

	areasCenters := []struct{ x, y int }{}
	for y := 0; y < m.Height*m.TileHeight+h/2; y += h / 2 {
		for x := 0; x < m.Width*m.TileWidth+w/2; x += w / 2 {
			areasCenters = append(areasCenters, struct{ x, y int }{x: x, y: y})
		}
	}

	rectsLayer := m.getLayerByName("collision-rects")
	if rectsLayer == nil {
		return
	}

	areaCenters := make([]MapObject, 0, len(areasCenters))
	for i, ac := range areasCenters {
		propName := fmt.Sprintf("area_%d", i+1)
		areaCenters = append(areaCenters, MapObject{
			Id:       i + 1,
			Name:     propName,
			Type:     "area-center",
			Rotation: 0,
			Visible:  true,
			Width:    0,
			Height:   0,
			X:        float64(ac.x),
			Y:        float64(ac.y),
		})
	}

	m.Layers = append(m.Layers, MapLayer{
		Data:    nil,
		Objects: areaCenters,
		Name:    "area-centers",
		Width:   0,
		Height:  0,
		Type:    "objectgroup",
		X:       0,
		Y:       0,
	})

	for i, rect := range rectsLayer.Objects {
		for ai, ac := range areasCenters {
			isRelevant := false
			// check if any corner is within area
			if abs(int(rect.X)-ac.x) <= w && abs(int(rect.Y)-ac.y) <= h ||
				abs(int(rect.X)+int(rect.Width)-ac.x) <= w && abs(int(rect.Y)+int(rect.Height)-ac.y) <= h {
				isRelevant = true
			}
			// check if rect spans area center horizontally or vertically
			if (int(rect.Width) > w && abs(int(rect.Y)-ac.y) <= h) ||
				(int(rect.Height) > h && abs(int(rect.X)-ac.x) <= w) {
				isRelevant = true
			}
			if isRelevant {
				propName := fmt.Sprintf("area_%d", ai+1)
				rectsLayer.Objects[i].Properties = append(rectsLayer.Objects[i].Properties, MapProperty{
					Name:  propName,
					Type:  "bool",
					Value: true,
				})
			}
		}
	}
}

func (m *Map) buildBlockedGrid() {
	wallLayer := m.getLayerByName("walls")
	if wallLayer == nil {
		return
	}
	abyssLayer := m.getLayerByName("abyss")

	m.gridWidth = wallLayer.Width
	m.gridHeight = wallLayer.Height
	m.blockedGrid = make([]bool, m.gridWidth*m.gridHeight)

	for y := 0; y < m.gridHeight; y++ {
		for x := 0; x < m.gridWidth; x++ {
			idx := x + y*m.gridWidth
			if idx >= len(wallLayer.Data) {
				continue
			}
			if t := wallLayer.Data[idx]; t != 0 {
				if props, ok := m.TilesPropertiesHash[t]; ok {
					if v, ok2 := props["absorbs_light"].(bool); ok2 && v {
						m.blockedGrid[idx] = true
						continue
					}
				}
			}
			if abyssLayer != nil && idx < len(abyssLayer.Data) {
				if t := abyssLayer.Data[idx]; t != 0 {
					if props, ok := m.TilesPropertiesHash[t]; ok {
						if v, ok2 := props["collides"].(bool); ok2 && v {
							m.blockedGrid[idx] = true
						}
					}
				}
			}
		}
	}
}

func (m *Map) isTileBlockedForMonster(tileX, tileY int) bool {
	if tileX < 0 || tileX >= m.gridWidth || tileY < 0 || tileY >= m.gridHeight {
		return true
	}
	idx := tileX + tileY*m.gridWidth
	if idx >= len(m.blockedGrid) {
		return true
	}
	return m.blockedGrid[idx]
}

// findPath runs A* from start tile to goal tile and returns a slice of pixel-center
// waypoints (not including the start position). Returns nil if no path exists.
func (m *Map) findPath(startTX, startTY, goalTX, goalTY int) []Point {
	if startTX == goalTX && startTY == goalTY {
		return nil
	}

	type node struct {
		tx, ty, g, f int
		parent       *node
	}

	heuristic := func(tx, ty int) int {
		dx := tx - goalTX
		dy := ty - goalTY
		if dx < 0 {
			dx = -dx
		}
		if dy < 0 {
			dy = -dy
		}
		return dx + dy
	}

	open := []*node{{tx: startTX, ty: startTY, g: 0, f: heuristic(startTX, startTY)}}
	closed := make(map[int]bool, 256)
	key := func(tx, ty int) int { return tx + ty*m.gridWidth }

	for len(open) > 0 {
		bestIdx := 0
		for i, n := range open {
			if n.f < open[bestIdx].f {
				bestIdx = i
			}
		}
		cur := open[bestIdx]
		open[bestIdx] = open[len(open)-1]
		open = open[:len(open)-1]

		k := key(cur.tx, cur.ty)
		if closed[k] {
			continue
		}
		closed[k] = true

		if cur.tx == goalTX && cur.ty == goalTY {
			var path []Point
			for n := cur; n.tx != startTX || n.ty != startTY; n = n.parent {
				path = append(path, Point{
					X: n.tx*m.TileWidth + m.TileWidth/2,
					Y: n.ty*m.TileHeight + m.TileHeight/2,
				})
			}
			for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
				path[i], path[j] = path[j], path[i]
			}
			return path
		}

		if len(closed) > 4096 {
			break
		}

		for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
			nx, ny := cur.tx+d[0], cur.ty+d[1]
			if m.isTileBlockedForMonster(nx, ny) || closed[key(nx, ny)] {
				continue
			}
			g := cur.g + 1
			open = append(open, &node{tx: nx, ty: ny, g: g, f: g + heuristic(nx, ny), parent: cur})
		}
	}

	return nil
}

// buildVisibilityColliders precomputes the static collision rectangles used for
// line-of-sight checks. The collision-rects layer never changes after map load,
// so caching the slice avoids rebuilding and reallocating it on every
// isVisible() call (which runs per monster × per player every AI tick).
func (m *Map) buildVisibilityColliders() {
	visLayer := m.getLayerByName("collision-rects")
	if visLayer == nil {
		return
	}

	m.visibilityColliders = make([]Rectangle, 0, len(visLayer.Objects))
	for _, obj := range visLayer.Objects {
		m.visibilityColliders = append(m.visibilityColliders, Rectangle{
			X:      int(obj.X),
			Y:      int(obj.Y),
			Width:  int(obj.Width),
			Height: int(obj.Height),
		})
	}
}

func (m *Map) getVisibilityColliders() []Rectangle {
	return m.visibilityColliders
}
