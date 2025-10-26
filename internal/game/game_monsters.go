package game

import (
	"time"
)

const period = time.Second / 5

const tileSize = 32

const skeletonAttackDuration = time.Second / 2
const archerAttackCooldown = time.Second / 2
const archerAttackDelay = 200 * time.Millisecond
const archerAttackDuration = 600 * time.Millisecond

const demonAttackFirecircleCooldown = 5 * time.Second
const demonAttackLightningCooldown = 6 * time.Second
const demonAttackCooldown = 2 * time.Second
const demonAttackDelay = 300 * time.Millisecond
const demonAttackDuration = time.Second

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
				case monsterKindDemon:
					g.intellectDemon(mon)
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
		if distance < minDistance &&
			distance <= 30*tileSize &&
			g.isVisible(mon.x, mon.y, player.x, player.y) {
			minDistance = distance
			closestPlayer = player
		}
	}

	if closestPlayer == nil {
		return
	}

	// Attack
	if mon.attackStartedAt.IsZero() {
		mon.attackStartedAt = time.Now()
		mon.isAttacking = true
		mon.attacked = false
		mon.direction = getDirection(mon.x, mon.y, closestPlayer.x, closestPlayer.y)
	} else if time.Since(mon.attackStartedAt) >= archerAttackCooldown {
		mon.attackStartedAt = time.Time{}
	} else if time.Since(mon.attackStartedAt) >= archerAttackDuration {
		mon.isAttacking = false
	} else if time.Since(mon.attackStartedAt) >= archerAttackDelay && !mon.attacked {
		mon.attacked = true
		g.broadcastEventFunc(ArrowEvent{
			ClientID:  0,
			MonsterID: mon.id,
			X1:        mon.x,
			Y1:        mon.y,
			X2:        closestPlayer.x,
			Y2:        closestPlayer.y,
		})
	}
}

func (g *Game) intellectDemon(mon *Monster) {
	closestPlayers := make([]*Player, 0)
	hasOneOnDirectLines := false

	for _, player := range g.players {
		if player.hp <= 0 {
			continue
		}

		distance := getDistance(mon.x, mon.y, player.x, player.y)
		if distance <= 30*tileSize &&
			g.isVisible(mon.x, mon.y, player.x, player.y) {
			closestPlayers = append(closestPlayers, player)
			if abs(player.x-mon.x) < tileSize || abs(player.y-mon.y) < tileSize {
				hasOneOnDirectLines = true
			}
		}
	}

	if time.Since(mon.lightningStartedAt) >= demonAttackLightningCooldown {
		mon.lightningStartedAt = time.Now()
		g.broadcastEventFunc(DemonLightningEvent{
			MonsterID: mon.id,
			X:         mon.x,
			Y:         mon.y,
		})
	}

	if hasOneOnDirectLines && time.Since(mon.firecircleStartedAt) >= demonAttackFirecircleCooldown {
		mon.firecircleStartedAt = time.Now()
		g.broadcastEventFunc(FireCircleEvent{
			ClientID:  0,
			MonsterID: mon.id,
			X:         mon.x,
			Y:         mon.y,
		})
	}

	// Attack
	if mon.attackStartedAt.IsZero() {
		if len(closestPlayers) == 0 {
			return
		}

		if len(closestPlayers) > 0 {
			mon.attackStartedAt = time.Now()
			mon.isAttacking = true
		}
	} else if !mon.attacked && time.Since(mon.attackStartedAt) >= demonAttackDelay {
		for _, closestPlayer := range closestPlayers {
			g.broadcastEventFunc(DemonFireballEvent{
				ClientID:  0,
				MonsterID: mon.id,
				X1:        mon.x,
				Y1:        mon.y,
				X2:        closestPlayer.x,
				Y2:        closestPlayer.y,
			})
		}
		if len(closestPlayers) > 0 {
			mon.attacked = true
		}
	} else if time.Since(mon.attackStartedAt) >= demonAttackCooldown {
		mon.attackStartedAt = time.Time{}
		mon.attacked = false
	} else if time.Since(mon.attackStartedAt) > demonAttackDuration {
		mon.isAttacking = false
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
		if distance < minDistance &&
			distance <= 10*tileSize &&
			g.isVisible(mon.x, mon.y, player.x, player.y) {
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

func (g *Game) isVisible(x1, y1, x2, y2 int) bool {
	colliders := g.gameMap.getVisibilityColliders()
	for _, col := range colliders {
		if lineIntersectsRect(x1, y1, x2, y2, col.X, col.Y, col.Width, col.Height) {
			return false
		}
	}

	return true
}

func getDirection(x1, y1, x2, y2 int) string {
	if abs(x2-x1) > abs(y2-y1) {
		if x2 > x1 {
			return "right"
		}

		return "left"
	}

	if y2 > y1 {
		return "down"
	}

	return "up"
}

func lineIntersectsRect(x1, y1, x2, y2, rx, ry, rw, rh int) bool {
	// Check if either endpoint is inside the rectangle
	if pointInRect(x1, y1, rx, ry, rw, rh) || pointInRect(x2, y2, rx, ry, rw, rh) {
		return true
	}

	// Check for intersection with each edge of the rectangle
	if linesIntersect(x1, y1, x2, y2, rx, ry, rx+rw, ry) || // Top edge
		linesIntersect(x1, y1, x2, y2, rx+rw, ry, rx+rw, ry+rh) || // Right edge
		linesIntersect(x1, y1, x2, y2, rx+rw, ry+rh, rx, ry+rh) || // Bottom edge
		linesIntersect(x1, y1, x2, y2, rx, ry+rh, rx, ry) { // Left edge
		return true
	}

	return false
}

func pointInRect(px, py, rx, ry, rw, rh int) bool {
	return px >= rx && px <= rx+rw && py >= ry && py <= ry+rh
}

// Check if two line segments (x1,y1)-(x2,y2) and (x3,y3)-(x4,y4) intersect
func linesIntersect(x1, y1, x2, y2, x3, y3, x4, y4 int) bool {
	// Helper: compute orientation of the ordered triplet (p1, p2, p3)
	// 0 → collinear
	// 1 → clockwise
	// 2 → counterclockwise
	orientation := func(x1, y1, x2, y2, x3, y3 int) int {
		val := (y2-y1)*(x3-x2) - (x2-x1)*(y3-y2)
		if val == 0 {
			return 0
		}
		if val > 0 {
			return 1
		}
		return 2
	}

	// Helper: check if point (x3,y3) lies on the segment (x1,y1)-(x2,y2)
	onSegment := func(x1, y1, x2, y2, x3, y3 int) bool {
		return x3 <= max(x1, x2) && x3 >= min(x1, x2) &&
			y3 <= max(y1, y2) && y3 >= min(y1, y2)
	}

	// Find orientations for the 4 combinations
	o1 := orientation(x1, y1, x2, y2, x3, y3)
	o2 := orientation(x1, y1, x2, y2, x4, y4)
	o3 := orientation(x3, y3, x4, y4, x1, y1)
	o4 := orientation(x3, y3, x4, y4, x2, y2)

	// General case: segments intersect if orientations differ
	if o1 != o2 && o3 != o4 {
		return true
	}

	// Special cases: check if points are collinear and lie on the other segment
	if o1 == 0 && onSegment(x1, y1, x2, y2, x3, y3) {
		return true
	}
	if o2 == 0 && onSegment(x1, y1, x2, y2, x4, y4) {
		return true
	}
	if o3 == 0 && onSegment(x3, y3, x4, y4, x1, y1) {
		return true
	}
	if o4 == 0 && onSegment(x3, y3, x4, y4, x2, y2) {
		return true
	}

	// Otherwise, no intersection
	return false
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
