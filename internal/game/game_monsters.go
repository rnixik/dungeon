package game

import (
	"time"
)

const period = time.Second / 5

const tileSize = 16

const skeletonAttackDuration = time.Second / 2
const archerAttackCooldown = time.Second / 2
const archerAttackDelay = 200 * time.Millisecond
const archerAttackDuration = 600 * time.Millisecond

const demonAttackFirecircleCooldown = 5 * time.Second
const demonAttackLightningCooldown = 6 * time.Second
const demonAttackCooldown = 2 * time.Second
const demonAttackDelay = 300 * time.Millisecond
const demonAttackDuration = time.Second

const golemAttackCooldown = 4 * time.Second
const golemAttackDelay = 500 * time.Millisecond // 5th frame at 8fps (4 × 125ms)
const golemAttackDuration = 1000 * time.Millisecond
const golemAttackRadius = 3 * tileSize
const golemAttackDamage = 100

const spiderMeleeAttackDuration = 500 * time.Millisecond
const spiderMeleeAttackDamage = 25
const spiderWebCooldown = 10 * time.Second
const spiderWebAttackDuration = 700 * time.Millisecond
const spiderWebRange = 15 * tileSize

const jellyAttackDuration = 1500 * time.Millisecond
const jellyAttackDelay = 400 * time.Millisecond
const jellyHitSlowDuration = 3000 // ms, sent to client

// moveTowardPlayer updates pathfinding state and moves mon toward player.
func (g *Game) moveTowardPlayer(mon *Monster, player *Player) {
	goalTX := player.x / tileSize
	goalTY := player.y / tileSize
	if len(mon.path) == 0 || mon.pathGoalTX != goalTX || mon.pathGoalTY != goalTY {
		mon.path = g.gameMap.findPath(mon.x/tileSize, mon.y/tileSize, goalTX, goalTY)
		mon.pathGoalTX = goalTX
		mon.pathGoalTY = goalTY
		if len(mon.path) > 0 {
			mon.moveToX = mon.path[0].X
			mon.moveToY = mon.path[0].Y
		}
	}
	mon.isMoving = len(mon.path) > 0
	mon.direction = getDirection(mon.x, mon.y, player.x, player.y)
}

// tickAttack advances the attack FSM: fires once at delay, holds animation until duration, resets at cooldown.
func tickAttack(mon *Monster, delay, duration, cooldown time.Duration, onFire func()) {
	elapsed := time.Since(mon.attackStartedAt)
	if elapsed >= cooldown {
		mon.attackStartedAt = time.Time{}
		mon.attacked = false
		return
	}
	if elapsed < duration {
		mon.isAttacking = true
	} else {
		mon.isAttacking = false
	}
	if !mon.attacked && elapsed >= delay {
		mon.attacked = true
		onFire()
	}
}

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
				case monsterKindGolem:
					g.intellectGolem(mon)
				case monsterKindSpider:
					g.intellectSpider(mon)
				case monsterKindJelly, monsterKindJellySmall, monsterKindJellyMicro:
					g.intellectJelly(mon)
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

	if mon.attackStartedAt.IsZero() {
		mon.attackStartedAt = time.Now()
		mon.isAttacking = true
		mon.attacked = false
		mon.direction = getDirection(mon.x, mon.y, closestPlayer.x, closestPlayer.y)
	} else {
		tickAttack(mon, archerAttackDelay, archerAttackDuration, archerAttackCooldown, func() {
			g.broadcastEventFunc(ArrowEvent{
				ClientID:  0,
				MonsterID: mon.id,
				X1:        mon.x,
				Y1:        mon.y,
				X2:        closestPlayer.x,
				Y2:        closestPlayer.y,
			})
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

	var lightningTarget *Player
	minLightningDist := 1000000
	for _, p := range closestPlayers {
		d := getDistance(mon.x, mon.y, p.x, p.y)
		if d < minLightningDist {
			minLightningDist = d
			lightningTarget = p
		}
	}
	if lightningTarget != nil && time.Since(mon.lightningStartedAt) >= demonAttackLightningCooldown {
		mon.lightningStartedAt = time.Now()
		g.broadcastEventFunc(DemonLightningEvent{
			MonsterID: mon.id,
			X:         mon.x,
			Y:         mon.y,
			TargetX:   lightningTarget.x,
			TargetY:   lightningTarget.y,
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

	if mon.attackStartedAt.IsZero() {
		if len(closestPlayers) == 0 {
			return
		}
		mon.attackStartedAt = time.Now()
		mon.isAttacking = true
	} else {
		tickAttack(mon, demonAttackDelay, demonAttackDuration, demonAttackCooldown, func() {
			for _, p := range closestPlayers {
				g.broadcastEventFunc(DemonFireballEvent{
					ClientID:  0,
					MonsterID: mon.id,
					X1:        mon.x,
					Y1:        mon.y,
					X2:        p.x,
					Y2:        p.y,
				})
			}
		})
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
			distance <= 20*tileSize &&
			g.isVisible(mon.x, mon.y, player.x, player.y) {
			minDistance = distance
			closestPlayer = player
		}
	}

	mon.isAttacking = false

	if closestPlayer == nil {
		mon.isMoving = false
		mon.path = nil
		return
	}

	if minDistance <= tileSize {
		mon.isMoving = false
		mon.path = nil
		mon.isAttacking = true
		if mon.attackStartedAt.IsZero() {
			mon.attackStartedAt = time.Now()
		} else if time.Since(mon.attackStartedAt) >= skeletonAttackDuration {
			mon.attackStartedAt = time.Time{}
			g.hitPlayerUnsafe(closestPlayer.client.ID(), 30)
		}
		return
	}

	g.moveTowardPlayer(mon, closestPlayer)
	if mon.isMoving && minDistance <= tileSize*2 {
		mon.isAttacking = true
	}
}

func (g *Game) intellectGolem(mon *Monster) {
	var closestPlayer *Player
	minDistance := 1000000
	for _, player := range g.players {
		if player.hp <= 0 {
			continue
		}
		distance := getDistance(mon.x, mon.y, player.x, player.y)
		if distance < minDistance && distance <= 25*tileSize {
			minDistance = distance
			closestPlayer = player
		}
	}

	mon.isAttacking = false

	if closestPlayer == nil {
		mon.isMoving = false
		mon.path = nil
		return
	}

	if mon.attackStartedAt.IsZero() {
		if minDistance <= golemAttackRadius {
			mon.attackStartedAt = time.Now()
			mon.isAttacking = true
			mon.attacked = false
			mon.isMoving = false
			mon.path = nil
		}
	} else {
		tickAttack(mon, golemAttackDelay, golemAttackDuration, golemAttackCooldown, func() {
			for _, player := range g.players {
				if player.hp <= 0 {
					continue
				}
				if getDistance(mon.x, mon.y, player.x, player.y) <= golemAttackRadius {
					g.hitPlayerUnsafe(player.client.ID(), golemAttackDamage)
				}
			}
			g.broadcastEventFunc(GolemSlamEvent{
				MonsterID: mon.id,
				X:         mon.x,
				Y:         mon.y,
				Radius:    golemAttackRadius,
			})
		})
	}

	if !mon.isAttacking {
		g.moveTowardPlayer(mon, closestPlayer)
	}
}

func (g *Game) intellectSpider(mon *Monster) {
	var closestPlayer *Player
	minDistance := 1000000
	for _, player := range g.players {
		if player.hp <= 0 {
			continue
		}
		distance := getDistance(mon.x, mon.y, player.x, player.y)
		if distance < minDistance && distance <= 20*tileSize {
			minDistance = distance
			closestPlayer = player
		}
	}

	mon.isAttacking = false

	if closestPlayer == nil {
		mon.isMoving = false
		mon.path = nil
		return
	}

	// Initialize web cooldown on first contact so spider waits before first throw
	if mon.webStartedAt.IsZero() {
		mon.webStartedAt = time.Now()
	}

	// Web throw with 10s cooldown
	if minDistance <= spiderWebRange && time.Since(mon.webStartedAt) >= spiderWebCooldown {
		mon.webStartedAt = time.Now()
		g.broadcastEventFunc(SpiderWebEvent{
			MonsterID: mon.id,
			X:         closestPlayer.x,
			Y:         closestPlayer.y,
		})
	}

	// Play attack animation briefly after web throw
	if !mon.webStartedAt.IsZero() && time.Since(mon.webStartedAt) < spiderWebAttackDuration {
		mon.isAttacking = true
	}

	// Melee attack when adjacent
	if minDistance <= tileSize {
		mon.isMoving = false
		mon.path = nil
		mon.isAttacking = true
		if mon.attackStartedAt.IsZero() {
			mon.attackStartedAt = time.Now()
		} else if time.Since(mon.attackStartedAt) >= spiderMeleeAttackDuration {
			mon.attackStartedAt = time.Time{}
			g.hitPlayerUnsafe(closestPlayer.client.ID(), spiderMeleeAttackDamage)
		}
		return
	}
	mon.attackStartedAt = time.Time{}

	g.moveTowardPlayer(mon, closestPlayer)
}

func (g *Game) intellectJelly(mon *Monster) {
	var closestPlayer *Player
	minDistance := 1000000
	for _, player := range g.players {
		if player.hp <= 0 {
			continue
		}
		distance := getDistance(mon.x, mon.y, player.x, player.y)
		if distance < minDistance && distance <= 20*tileSize {
			minDistance = distance
			closestPlayer = player
		}
	}

	mon.isAttacking = false

	if closestPlayer == nil {
		mon.isMoving = false
		mon.path = nil
		return
	}

	if minDistance <= tileSize {
		mon.isMoving = false
		mon.path = nil
		mon.isAttacking = true
		if mon.attackStartedAt.IsZero() {
			mon.attackStartedAt = time.Now()
			mon.attacked = false
		} else {
			tickAttack(mon, jellyAttackDelay, jellyAttackDuration, jellyAttackDuration, func() {
				g.hitPlayerUnsafe(closestPlayer.client.ID(), mon.damage)
				closestPlayer.client.SendEvent(JellyHitSlowEvent{
					Duration:    jellyHitSlowDuration,
					SlowPercent: 80,
				})
			})
		}
		return
	}
	mon.attackStartedAt = time.Time{}
	mon.attacked = false

	g.moveTowardPlayer(mon, closestPlayer)
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
