package game

import (
	"encoding/json"
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
	Properties map[string]any `json:"properties"`
}

type MapLayer struct {
	Data    []int       `json:"data"`
	Objects []MapObject `json:"objects"`
	Name    string      `json:"name"`
	Width   int         `json:"width"`
	Height  int         `json:"height"`
	Type    string      `json:"type"`
	X       int         `json:"x"`
	Y       int         `json:"y"`
}

type MapTileset struct {
	Columns     int    `json:"columns"`
	Firstgid    int    `json:"firstgid"`
	Image       string `json:"image"`
	Imageheight int    `json:"imageheight"`
	Imagewidth  int    `json:"imagewidth"`
	Margin      int    `json:"margin"`
	Name        string `json:"name"`
	Spacing     int    `json:"spacing"`
	Tilecount   int    `json:"tilecount"`
	Tileheight  int    `json:"tileheight"`
	Tilewidth   int    `json:"tilewidth"`
}

type Map struct {
	Width       int          `json:"width"`
	Height      int          `json:"height"`
	Layers      []MapLayer   `json:"layers"`
	TileWidth   int          `json:"tilewidth"`
	TileHeight  int          `json:"tileheight"`
	Tilesets    []MapTileset `json:"tilesets"`
	Orientation string       `json:"orientation"`
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
