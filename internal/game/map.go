package game

import (
	"encoding/json"
	"fmt"
	"os"
)

type MapObject struct {
	Id         int            `json:"id"`
	Name       string         `json:"name"`
	Type       string         `json:"type"`
	Rotation   float64        `json:"rotation"`
	Visible    bool           `json:"visible"`
	Width      float64        `json:"width"`
	Height     float64        `json:"height"`
	X          float64        `json:"x"`
	Y          float64        `json:"y"`
	Properties map[string]any `json:"properties,omitempty"`
	Point      bool           `json:"point,omitempty"`
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

type Map struct {
	Compression int          `json:"compressionlevel"`
	Infinite    bool         `json:"infinite"`
	Width       int          `json:"width"`
	Height      int          `json:"height"`
	Layers      []MapLayer   `json:"layers"`
	TileWidth   int          `json:"tilewidth"`
	TileHeight  int          `json:"tileheight"`
	Tilesets    []MapTileset `json:"tilesets"`
	Orientation string       `json:"orientation"`
	RenderOrder string       `json:"renderorder"`
	Type        string       `json:"type"`
	Version     string       `json:"version"`
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

	err = m.addLayerWithCollisionRectangles()
	if err != nil {
		return nil, err
	}

	return &m, err
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
		return t != 0 // assuming non-zero tile means solid
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
			Properties: map[string]any{
				"collides": true,
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

/* js function
function getBigRectsFromWallLayer(layer) {
    const mapW = layer.layer.width;
    const mapH = layer.layer.height;
    const tw = layer.tilemap.tileWidth;
    const th = layer.tilemap.tileHeight;

    const isSolidAt = (x, y) => {
        const t = layer.getTileAt(x, y);
        return !!t && (t.collides === true || t.properties?.collides === true);
    };

    // 1) horizontal runs per row
    const runs = Array.from({ length: mapH }, () => []);
    for (let y = 0; y < mapH; y++) {
        let x = 0;
        while (x < mapW) {
            if (!isSolidAt(x, y)) { x++; continue; }
            const x0 = x;
            while (x < mapW && isSolidAt(x, y)) x++;
            runs[y].push({ x: x0, w: x - x0 });
        }
    }

    // 2) vertical merge of identical runs
    const rects = [];
    const used = runs.map(row => row.map(() => false));

    for (let y = 0; y < mapH; y++) {
        for (let i = 0; i < runs[y].length; i++) {
            if (used[y][i]) continue;
            const { x: rx, w: rw } = runs[y][i];
            let h = 1;
            // try to extend downwards while the exact same run exists and not used
            let yy = y + 1;
            while (yy < mapH) {
                let foundIdx = -1;
                for (let j = 0; j < runs[yy].length; j++) {
                    if (!used[yy][j] && runs[yy][j].x === rx && runs[yy][j].w === rw) { foundIdx = j; break; }
                }
                if (foundIdx === -1) break;
                used[yy][foundIdx] = true;
                h++;
                yy++;
            }
            used[y][i] = true;

            const t0 = layer.getTileAt(rx, y);
            rects.push(new Rectangle(t0.getLeft(), t0.getTop(), rw * tw, h * th));
        }
    }

    return rects;
}
*/
