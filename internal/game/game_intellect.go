package game

import "time"

const period = time.Second / 2

const tileSize = 32

func (g *Game) startIntellect() {
	ticker := time.NewTicker(period)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if g.isGameEnded() {
				return
			}

			g.mutex.Lock()

			for _, mon := range g.monsters {
				if mon.hp <= 0 {
					continue
				}
				for _, player := range g.players {
					if player.hp <= 0 {
						continue
					}
					if getDistance(mon.x, mon.y, player.x, player.y) <= 10*tileSize {
						g.broadcastEventFunc(ArrowEvent{
							ClientID:  0,
							MonsterID: mon.id,
							X1:        mon.x,
							Y1:        mon.y,
							X2:        player.x,
							Y2:        player.y,
						})
					}
				}
			}

			g.mutex.Unlock()
		}
	}
}

func getDistance(x1, y1, x2, y2 int) int {
	dx := x2 - x1
	dy := y2 - y1

	return abs(dx) + abs(dy)
}

func abs(a int) int {
	if a < 0 {
		return -a
	}

	return a
}
