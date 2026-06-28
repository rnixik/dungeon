package game

import (
	"math/rand"
	"time"
)

const objectsPeriod = time.Second / 10
const trapSize = 64

// chestMonsterRange is how close a living monster must be (to the opener or the
// chest) to block the chest from being opened.
const chestMonsterRange = tileSize * 3

func (g *Game) startObjectsLoop() {
	ticker := time.NewTicker(objectsPeriod)
	defer ticker.Stop()
	deltaTime := objectsPeriod.Seconds()

	for {
		select {
		case <-ticker.C:
			if g.isGameEnded() {
				return
			}

			g.mutex.Lock()

			// Update objects
			for _, obj := range g.objects {
				switch obj.Kind {
				case objectKindChest:
					g.tickChest(obj)
				case objectKindTrigger:
					g.tickTrigger(obj)
				}
			}

			// Update traps
			g.tickTraps(deltaTime)

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
			// A chest cannot be opened while a living monster is near the
			// opener or near the chest itself.
			if g.monsterNearUnsafe(player.x, player.y) || g.monsterNearUnsafe(obj.X, obj.Y) {
				continue
			}

			obj.State = "open"
			g.revealPlayerUnsafe(player)
			g.broadcastEventFunc(ChestOpenEvent{ObjectID: obj.ID})

			// A chest may carry a loot item (Tiled "item" property). The opener
			// receives one of that item. Unknown kinds are ignored.
			if itemKind, ok := obj.PropertiesMap["item"].(string); ok && itemKind != "" {
				g.giveItemToPlayerUnsafe(player, itemKind)
			}

			// A debug chest (alwaysCurse property) curses the opener with 100%
			// chance and ignores the cultist cap. A normal chest curses by chance,
			// only while the cap (a third of the players) is not reached.
			alwaysCurse, _ := obj.PropertiesMap["alwaysCurse"].(bool)
			if !player.isCultist {
				if alwaysCurse {
					g.makePlayerCultistUnsafe(player)
				} else if g.cultistCountUnsafe() < g.maxCultistsAllowedUnsafe() &&
					rand.Float64() < cultistCurseChance {
					g.makePlayerCultistUnsafe(player)
				}
			}

			// Each chest yields the next uncollected key. With more chests than
			// keys, opening any full set of chests collects every key.
			for _, number := range []string{"1", "2", "3"} {
				if !g.keysCollected[number] {
					g.keysCollected[number] = true
					g.broadcastEventFunc(KeyCollectedEvent{Number: number})
					break
				}
			}

			allKeysCollected := g.keysCollected["1"] && g.keysCollected["2"] && g.keysCollected["3"]

			if allKeysCollected && g.demonWasSpawned == false {
				g.spawnDemonUnsafe()
			}

			// Only the opener interacts with the chest this tick.
			return
		}
	}
}

// monsterNearUnsafe reports whether a living monster is within chestMonsterRange
// of the given point. Caller must hold g.mutex.
func (g *Game) monsterNearUnsafe(x, y int) bool {
	for _, mon := range g.monsters {
		if mon.hp <= 0 {
			continue
		}
		if getDistance(x, y, mon.x, mon.y) <= chestMonsterRange {
			return true
		}
	}
	return false
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
						// Activate trap using new trap system
						trapID := targetObj.PropertiesMap["trapId"]
						if trapIDStr, ok := trapID.(string); ok {
							if trap, exists := g.traps[trapIDStr]; exists {
								trap.Activate()
								g.broadcastEventFunc(TrapStateChangedEvent{
									TrapID: trap.ID,
									State:  trap.State,
									X:      trap.Params.X,
									Y:      trap.Params.Y,
									Frame:  trap.GetCurrentFrame(),
								})
							}
						}
					}
				}
			}
		}
	}
}

func (g *Game) tickTraps(deltaTime float64) {
	for _, trap := range g.traps {
		stateChanged, newState := trap.Tick(deltaTime)

		if stateChanged {
			g.broadcastEventFunc(TrapStateChangedEvent{
				TrapID: trap.ID,
				State:  newState,
				X:      trap.Params.X,
				Y:      trap.Params.Y,
				Frame:  trap.GetCurrentFrame(),
			})
		}

		if trap.IsActive() {
			for _, mon := range g.monsters {
				if mon.hp <= 0 || trap.LastDamagedMonsters[mon.id] {
					continue
				}
				if mon.x >= trap.Params.X && mon.x < trap.Params.X+trapSize &&
					mon.y >= trap.Params.Y && mon.y < trap.Params.Y+trapSize {
					trap.LastDamagedMonsters[mon.id] = true
					g.hitMonsterUnsafe(0, mon.id, trap.Params.Damage)
				}
			}
		}
	}
}
