package game

import "time"

const period = time.Second / 5

const tileSize = 32

const skeletonAttackDuration = time.Second / 2
const archerAttackCooldown = time.Second / 2

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

				switch mon.kind {
				case monsterKindArcher:
					g.intellectArcher(mon)
				case monsterKindSkeleton:
					g.intellectSkeleton(mon)
				}

			}

			g.mutex.Unlock()
		}
	}
}

func (g *Game) intellectArcher(mon *Monster) {
	var closestPlayer *Player
	minDistance := 1000000

	for _, player := range g.players {
		if player.hp <= 0 {
			continue
		}

		distance := getDistance(mon.x, mon.y, player.x, player.y)
		if distance < minDistance && distance <= 10*tileSize {
			minDistance = distance
			closestPlayer = player
		}
	}

	if closestPlayer == nil {
		return
	}

	// Attack
	if mon.attackStartedAt.IsZero() {
		g.broadcastEventFunc(ArrowEvent{
			ClientID:  0,
			MonsterID: mon.id,
			X1:        mon.x,
			Y1:        mon.y,
			X2:        closestPlayer.x,
			Y2:        closestPlayer.y,
		})

		mon.attackStartedAt = time.Now()
	} else if time.Since(mon.attackStartedAt) >= archerAttackCooldown {
		mon.attackStartedAt = time.Time{}
	}

}

func (g *Game) intellectSkeleton(mon *Monster) {
	var closestPlayer *Player
	minDistance := 1000000
	for _, player := range g.players {
		if player.hp <= 0 {
			continue
		}

		distance := getDistance(mon.x, mon.y, player.x, player.y)
		if distance < minDistance && distance <= 10*tileSize {
			minDistance = distance
			closestPlayer = player
		}
	}

	mon.isMoving = false
	mon.isAttacking = false

	if closestPlayer == nil {
		return
	}

	if minDistance <= tileSize {
		// Attack
		mon.isAttacking = true
		if mon.attackStartedAt.IsZero() {
			mon.attackStartedAt = time.Now()
		} else if time.Since(mon.attackStartedAt) >= skeletonAttackDuration {
			mon.attackStartedAt = time.Time{}
			g.hitPlayerUnsafe(closestPlayer.client.ID(), 30)
		}
	} else {
		// Move towards player
		mon.isMoving = true
		mon.moveToX = closestPlayer.x
		mon.moveToY = closestPlayer.y
		if minDistance <= tileSize*1.5 {
			// start attack animation if close enough
			mon.isAttacking = true
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
