package game

import (
	"time"
)

const objectsPeriod = time.Second / 10

func (g *Game) startObjectsLoop() {
	ticker := time.NewTicker(objectsPeriod)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if g.isGameEnded() {
				return
			}

			g.mutex.Lock()

			for _, obj := range g.objects {
				switch obj.Kind {
				case objectKindChest:
					g.tickChest(obj)
				case objectKindTrigger:
					g.tickTrigger(obj)
				}

			}

			g.mutex.Unlock()
		}
	}
}

func (g *Game) tickChest(obj *Object) {
	if obj.State == "open" {
		return
	}

	for _, player := range g.players {
		if player.hp <= 0 {
			continue
		}

		distance := getDistance(obj.X, obj.Y, player.x, player.y)
		if distance <= tileSize*3 && g.isVisible(obj.X, obj.Y, player.x, player.y) {
			obj.State = "open"
			g.broadcastEventFunc(ChestOpenEvent{ObjectID: obj.ID})
			if g.keysCollected["3"] != true {
				g.keysCollected["3"] = true
				g.broadcastEventFunc(KeyCollectedEvent{Number: "3"})
			}

			allKeysCollected := g.keysCollected["1"] && g.keysCollected["2"] && g.keysCollected["3"]

			if allKeysCollected && g.demonWasSpawned == false {
				g.spawnDemonUnsafe()
			}
		}
	}
}

func (g *Game) tickTrigger(obj *Object) {
	if obj.State != "ready" {
		return
	}

	for _, player := range g.players {
		if player.hp <= 0 {
			continue
		}

		if pointInRect(player.x, player.y, obj.X, obj.Y, obj.Width, obj.Height) {
			obj.State = "activated"

			// replace tiles from special layer "replacements"
			tilesToUpdate := []TileData{}
			for tileIndex, tileID := range g.gameMap.getLayerByName("replacements").Data {
				tileX := (tileIndex % g.gameMap.Width) * tileSize
				tileY := (tileIndex / g.gameMap.Width) * tileSize
				if tileID > 0 && pointInRect(tileX, tileY, obj.X, obj.Y, obj.Width, obj.Height) {
					tilesToUpdate = append(tilesToUpdate, TileData{
						X:      tileX,
						Y:      tileY,
						TileID: tileID,
					})
				}
			}

			tileEvent := UpdateTilesEvent{
				LayerName: "floor",
				Tiles:     tilesToUpdate,
			}
			g.updateTilesEvents = append(g.updateTilesEvents, tileEvent)
			g.broadcastEventFunc(tileEvent)

			// find objects with target group
			for _, targetObj := range g.objects {
				if targetObj.PropertiesMap["group"] == obj.PropertiesMap["target"] {
					if targetObj.Kind == objectKindTrapArrow {
						x2 := targetObj.X
						y2 := targetObj.Y
						if targetObj.PropertiesMap["direction"] == "right" {
							x2 = targetObj.X + tileSize
						}
						if targetObj.PropertiesMap["direction"] == "left" {
							x2 = targetObj.X - tileSize
						}
						g.broadcastEventFunc(ArrowEvent{
							MonsterID: -1,
							X1:        targetObj.X,
							Y1:        targetObj.Y,
							X2:        x2,
							Y2:        y2,
						})
					}
					if targetObj.Kind == objectKindTrapSpikes {
						// find closes tile coordinate and spawn spikes there
						tileX := (targetObj.X / tileSize) * tileSize
						tileY := (targetObj.Y / tileSize) * tileSize

						event := SpawnSpikeEvent{
							X:          tileX,
							Y:          tileY,
							StartFrame: targetObj.PropertiesMap["frame"].(string),
						}

						g.spikeEvents = append(g.spikeEvents, event)
						g.broadcastEventFunc(event)
					}
				}
			}
		}
	}
}
