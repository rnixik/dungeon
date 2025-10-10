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
		if distance <= tileSize && g.isVisible(obj.X, obj.Y, player.x, player.y) {
			obj.State = "open"
			g.broadcastEventFunc(ChestOpenEvent{ObjectID: obj.ID})
		}
	}
}
