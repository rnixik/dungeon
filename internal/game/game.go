package game

import (
	"dungeon/internal/lobby"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"
)

const StatusStarted = "started"
const StatusEnded = "ended"

const ClassMage = "mage"
const ClassKnight = "knight"
const ClassRogue = "rogue"

const fireballDamage = 25
const swordDamage = 50
const positionsUpdateTickPeriod = time.Second / 60
const commonUpdateTickPeriod = time.Second / 3

const monsterKindArcher = "archer"
const monsterKindSkeleton = "skeleton"
const monsterKindDemon = "demon"

const attackCooldown = time.Second / 2
const attackShotArrowCooldown = time.Second / 4
const attackSwordCooldown = time.Second
const attackSwordDelay = time.Millisecond * 700

const objectKindChest = "chest"
const objectKindTrigger = "trigger"
const objectKindTrapArrow = "trap_arrow"
const objectKindTrapSpikes = "trap_spikes"

type Player struct {
	client         lobby.ClientPlayer
	class          string
	lastAttackTime time.Time
	color          string
	maxHp          int
	hp             int
	x              int
	y              int
	direction      string
	isMoving       bool
}

type Monster struct {
	id                  int
	kind                string
	hp                  int
	x                   int
	y                   int
	direction           string
	isMoving            bool
	isAttacking         bool
	attacked            bool
	attackStartedAt     time.Time
	moveToX             int
	moveToY             int
	firecircleStartedAt time.Time
	lightningStartedAt  time.Time
}

type Object struct {
	ID            int                    `json:"id"`
	Kind          string                 `json:"kind"`
	X             int                    `json:"x"`
	Y             int                    `json:"y"`
	Width         int                    `json:"width"`
	Height        int                    `json:"height"`
	State         string                 `json:"state"`
	PropertiesMap map[string]interface{} `json:"-"`
}

func newPlayer(client lobby.ClientPlayer) *Player {
	// Assign a random hex color to the player
	colorHex := fmt.Sprintf("0x%06x", rand.Intn(0xFFFFFF))

	classes := []string{ClassMage, ClassKnight, ClassRogue}
	class := classes[rand.Intn(len(classes))]

	currentMaxHP := 100
	switch class {
	case ClassMage:
		currentMaxHP = 150
	case ClassKnight:
		currentMaxHP = 250
	case ClassRogue:
		currentMaxHP = 200
	}

	return &Player{
		client:    client,
		class:     class,
		color:     colorHex,
		maxHp:     currentMaxHP,
		hp:        currentMaxHP,
		x:         120,
		y:         140,
		direction: "right",
		isMoving:  false,
	}
}

type Game struct {
	players            map[uint64]*Player
	status             string
	broadcastEventFunc func(event interface{})
	mutex              sync.Mutex
	statusMx           sync.Mutex
	room               *lobby.Room
	monsters           []*Monster
	objects            map[uint64]*Object
	gameMap            *Map
	demonWasSpawned    bool
	keysCollected      map[string]bool
}

func NewGame(playersClients []lobby.ClientPlayer, room *lobby.Room, broadcastEventFunc func(event interface{}), gameMap *Map) *Game {
	players := make(map[uint64]*Player, len(playersClients))
	for _, client := range playersClients {
		players[client.ID()] = newPlayer(client)
	}

	log.Printf("new game created by %s\n", playersClients[0].Nickname())

	return &Game{
		status:             StatusStarted,
		players:            players,
		broadcastEventFunc: broadcastEventFunc,
		mutex:              sync.Mutex{},
		room:               room,
		monsters:           []*Monster{},
		gameMap:            gameMap,
		objects:            make(map[uint64]*Object),
		keysCollected: map[string]bool{
			"1": true,
			"2": true,
			"3": false,
		},
	}
}

func (g *Game) DispatchGameCommand(client lobby.ClientPlayer, commandName string, commandData interface{}) {
	if g.isGameEnded() {
		return
	}

	eventDataJson, ok := commandData.(json.RawMessage)
	if !ok {
		log.Printf("cannot decode event data for event name = %s\n", commandName)
		return
	}

	switch commandName {
	case "PlayerMoveCommand":
		var c MoveCommand
		if err := json.Unmarshal(eventDataJson, &c); err != nil {
			log.Printf("cannot decode PlayerMoveCommand: %v\n", err)

			return
		}
		g.movePlayerTo(client.ID(), c.X, c.Y, c.Direction, c.IsMoving)
		break
	case "CastFireballCommand":
		var c CastFireballCommand
		if err := json.Unmarshal(eventDataJson, &c); err != nil {
			log.Printf("cannot decode CastFireballCommand: %v\n", err)

			return
		}
		g.castFireball(client.ID(), c.X, c.Y, c.Direction)
		break
	case "SwordAttackCommand":
		var c SwordAttackCommand
		if err := json.Unmarshal(eventDataJson, &c); err != nil {
			log.Printf("cannot decode SwordAttackCommand: %v\n", err)

			return
		}
		g.attackWithSword(client.ID(), c.X, c.Y, c.Direction)
		break
	case "ShootArrowCommand":
		var c ShootArrowCommand
		if err := json.Unmarshal(eventDataJson, &c); err != nil {
			log.Printf("cannot decode ShootArrowCommand: %v\n", err)

			return
		}
		g.shootArrow(client.ID(), c.X, c.Y, c.Direction)
		break
	case "HitPlayerCommand":
		var c HitPlayerCommand
		if err := json.Unmarshal(eventDataJson, &c); err != nil {
			log.Printf("cannot decode HitPlayerCommand: %v\n", err)

			return
		}
		g.mutex.Lock()
		g.hitPlayerUnsafe(c.TargetClientID, fireballDamage)
		g.mutex.Unlock()
		break
	case "HitMonsterCommand":
		var c HitMonsterCommand
		if err := json.Unmarshal(eventDataJson, &c); err != nil {
			log.Printf("cannot decode HitMonsterCommand: %v\n", err)

			return
		}
		g.hitMonster(c.OriginClientID, c.MonsterID, fireballDamage)
		break
	}
}

func (g *Game) OnClientRemoved(client lobby.ClientPlayer) {
	if g.isGameEnded() {
		return
	}
	log.Printf("client '%s' removed from game\n", client.Nickname())
	g.mutex.Lock()
	g.killPlayer(client.ID())
	g.mutex.Unlock()
}

func (g *Game) OnClientJoined(client lobby.ClientPlayer) {
	log.Printf("client '%s' joined game\n", client.Nickname())
	g.mutex.Lock()
	g.players[client.ID()] = newPlayer(client)
	g.mutex.Unlock()
	client.SendEvent(JoinToStartedGameEvent{GameData: g.getPlayerInitialGameData(g.players[client.ID()])})
}

func (g *Game) GetCommonInitialGameData() map[string]interface{} {
	return map[string]interface{}{}
}

func (g *Game) getPlayerInitialGameData(pl *Player) map[string]interface{} {
	return map[string]interface{}{
		"mapData":     g.gameMap,
		"gameObjects": g.objects,
		"playerData": PlayerStats{
			PlayerPosition: PlayerPosition{
				ClientID:  pl.client.ID(),
				X:         pl.x,
				Y:         pl.y,
				Direction: pl.direction,
				IsMoving:  pl.isMoving,
			},
			Class:    pl.class,
			Nickname: pl.client.Nickname(),
			Color:    pl.color,
			MaxHP:    pl.maxHp,
			HP:       pl.hp,
		},
		"keysCollected": g.keysCollected,
	}
}

func (g *Game) sendPlayerInitialGameData() {
	for _, p := range g.players {
		p.client.SendEvent(JoinToStartedGameEvent{GameData: g.getPlayerInitialGameData(p)})
	}
}

func (g *Game) StartMainLoop() {
	g.spawnInitialMonsters()
	g.spawnInitialObjects()
	g.sendPlayerInitialGameData()
	go g.startIntellect()
	go g.startObjectsLoop()
	tickerPositions := time.NewTicker(positionsUpdateTickPeriod)
	tickerCommon := time.NewTicker(commonUpdateTickPeriod)
	defer tickerPositions.Stop()
	defer tickerCommon.Stop()
	for {
		select {
		case <-tickerPositions.C:
			if g.isGameEnded() {
				return
			}

			g.mutex.Lock()

			g.moveMonstersUnsafe()

			p := make([]PlayerPosition, 0, len(g.players))
			for _, pl := range g.players {
				if !pl.isMoving {
					continue
				}
				p = append(p, PlayerPosition{
					ClientID:  pl.client.ID(),
					X:         pl.x,
					Y:         pl.y,
					Direction: pl.direction,
					IsMoving:  pl.isMoving,
				})
			}
			m := make([]MonsterPosition, 0, len(g.monsters))
			for _, mon := range g.monsters {
				if !mon.isMoving && !mon.isAttacking {
					continue
				}
				m = append(m, MonsterPosition{
					ID:          mon.id,
					X:           mon.x,
					Y:           mon.y,
					Direction:   mon.direction,
					IsMoving:    mon.isMoving,
					IsAttacking: mon.isAttacking,
				})
			}

			g.broadcastEventFunc(CreaturesPosUpdateEvent{
				Players:  p,
				Monsters: m,
			})

			g.mutex.Unlock()
		case <-tickerCommon.C:
			if g.isGameEnded() {
				return
			}
			g.mutex.Lock()

			p := make([]PlayerStats, 0, len(g.players))
			for _, pl := range g.players {
				p = append(p, PlayerStats{
					PlayerPosition: PlayerPosition{
						ClientID:  pl.client.ID(),
						X:         pl.x,
						Y:         pl.y,
						Direction: pl.direction,
						IsMoving:  pl.isMoving,
					},
					Class:    pl.class,
					Nickname: pl.client.Nickname(),
					Color:    pl.color,
					MaxHP:    pl.maxHp,
					HP:       pl.hp,
				})
			}
			m := make([]MonsterStats, 0, len(g.monsters))
			for _, mon := range g.monsters {
				m = append(m, MonsterStats{
					MonsterPosition: MonsterPosition{
						ID:          mon.id,
						X:           mon.x,
						Y:           mon.y,
						Direction:   mon.direction,
						IsMoving:    mon.isMoving,
						IsAttacking: mon.isAttacking,
					},
					Kind: mon.kind,
					HP:   mon.hp,
				})
			}

			g.broadcastEventFunc(CreaturesStatsUpdateEvent{
				Players:  p,
				Monsters: m,
			})

			g.mutex.Unlock()
		}
	}
}

func (g *Game) Status() string {
	g.statusMx.Lock()
	defer g.statusMx.Unlock()

	return g.status
}

func (g *Game) movePlayerTo(clientID uint64, x int, y int, direction string, isMoving bool) {
	g.mutex.Lock()
	defer g.mutex.Unlock()
	if p, ok := g.players[clientID]; ok {
		p.x = x
		p.y = y
		p.direction = direction
		p.isMoving = isMoving
	}
}

func (g *Game) castFireball(clientID uint64, x int, y int, direction string) {
	isDead := false

	var player *Player
	g.mutex.Lock()
	if p, ok := g.players[clientID]; ok {
		player = p
		if p.hp <= 0 {
			isDead = true
		}
	}
	g.mutex.Unlock()

	if player == nil {
		return
	}

	if isDead {
		return
	}

	if time.Since(player.lastAttackTime) < attackCooldown {
		return
	}
	player.lastAttackTime = time.Now()

	g.broadcastEventFunc(FireballEvent{
		ClientID:  clientID,
		X:         x,
		Y:         y,
		Direction: direction,
	})
}

func (g *Game) shootArrow(clientID uint64, x int, y int, direction string) {
	go func() {
		time.Sleep(time.Millisecond * 200)

		isDead := false

		var player *Player
		g.mutex.Lock()
		if p, ok := g.players[clientID]; ok {
			player = p
			if p.hp <= 0 {
				isDead = true
			}
		}
		g.mutex.Unlock()

		if player == nil {
			return
		}

		if isDead {
			return
		}

		if time.Since(player.lastAttackTime) < attackShotArrowCooldown {
			return
		}
		player.lastAttackTime = time.Now()

		const dispersion = 100.0
		vecX, vecY := getVectorFromDirection(direction)
		vecXDisp := vecX*1000 + (rand.Float64()*2-1)*dispersion
		vecYDisp := vecY*1000 + (rand.Float64()*2-1)*dispersion
		attackX, attackY := float64(x)+vecXDisp, float64(y)+vecYDisp

		g.broadcastEventFunc(ShootArrowEvent{
			ClientID: clientID,
			X1:       x + 20*int(vecX), // fix offset from player center
			Y1:       y + 20*int(vecY),
			X2:       int(attackX),
			Y2:       int(attackY),
			Velocity: 700,
		})
	}()
}

func (g *Game) attackWithSword(clientID uint64, x int, y int, direction string) {
	isDead := false

	var player *Player
	g.mutex.Lock()
	if p, ok := g.players[clientID]; ok {
		player = p
		if p.hp <= 0 {
			isDead = true
		}
	}
	g.mutex.Unlock()

	if player == nil {
		return
	}

	if isDead {
		return
	}

	if time.Since(player.lastAttackTime) < attackSwordCooldown {
		return
	}
	player.lastAttackTime = time.Now()

	g.broadcastEventFunc(SwordAttackPrepareEvent{
		ClientID:  clientID,
		X:         player.x,
		Y:         player.y,
		Direction: direction,
	})

	go func() {
		time.Sleep(attackSwordDelay)
		const length = 160
		const radius = 50

		g.mutex.Lock()
		if p, ok := g.players[clientID]; ok {
			player = p
			if p.hp <= 0 {
				isDead = true
			}
		}
		g.mutex.Unlock()

		if player == nil {
			return
		}
		if isDead {
			return
		}

		vecX, vecY := getVectorFromDirection(direction)
		attackX, attackY := player.x+int(vecX)*length, player.y+int(vecY)*length

		g.mutex.Lock()
		for _, p := range g.players {
			if p.client.ID() == clientID {
				continue
			}
			if (g.isSwordAttackHit(player.x, player.y, attackX, attackY, p.x, p.y, radius)) == false {
				continue
			}
			g.hitPlayerUnsafe(p.client.ID(), swordDamage)
		}
		for _, m := range g.monsters {
			if (g.isSwordAttackHit(player.x, player.y, attackX, attackY, m.x, m.y, radius)) == false {
				continue
			}
			g.hitMonsterUnsafe(m.id, swordDamage)
		}
		g.mutex.Unlock()

		g.broadcastEventFunc(SwordAttackEvent{
			ClientID:    clientID,
			X:           player.x,
			Y:           player.y,
			Direction:   direction,
			AttackLineX: attackX,
			AttackLineY: attackY,
		})
	}()
}

func (g *Game) hitPlayerUnsafe(targetClientID uint64, damage int) {
	if p, ok := g.players[targetClientID]; ok {
		if p.hp == 0 {
			return
		}

		p.hp -= damage
		if p.hp < 0 {
			p.hp = 0
		}

		g.broadcastEventFunc(DamageEvent{
			TargetPlayerId: targetClientID,
			Damage:         damage,
		})

		if p.hp == 0 {
			g.killPlayer(targetClientID)
		}
	}
}

func (g *Game) killPlayer(clientID uint64) {
	if p, ok := g.players[clientID]; ok {
		p.hp = 0

		g.broadcastEventFunc(PlayerDeathEvent{ClientID: clientID})
	}
}

func (g *Game) hitMonster(originClientID uint64, monsterID int, damage int) {
	g.mutex.Lock()
	defer g.mutex.Unlock()
	g.hitMonsterUnsafe(monsterID, damage)
}

func (g *Game) hitMonsterUnsafe(monsterID int, damage int) {
	for _, m := range g.monsters {
		if m.id == monsterID && m.hp > 0 {
			m.hp -= damage
			if m.hp < 0 {
				m.hp = 0
			}

			g.broadcastEventFunc(DamageEvent{
				TargetMonsterID: monsterID,
				Damage:          damage,
			})

			break
		}
	}
}

func (g *Game) endGame(winnerPlayerId uint64) {
	g.statusMx.Lock()
	g.status = StatusEnded
	g.statusMx.Unlock()

	g.broadcastEventFunc(EndGameEvent{WinnerPlayerId: winnerPlayerId})
	g.room.OnGameEnded()
}

func (g *Game) isGameEnded() bool {
	g.statusMx.Lock()
	defer g.statusMx.Unlock()

	return g.status == StatusEnded
}

func (g *Game) moveMonstersUnsafe() {
	for _, mon := range g.monsters {
		if mon.hp <= 0 {
			continue
		}

		moveSpeedPerTick := 2

		if mon.isMoving {
			if mon.x < mon.moveToX {
				mon.x += moveSpeedPerTick
				mon.direction = "right"
				if mon.x > mon.moveToX {
					mon.x = mon.moveToX
					mon.isMoving = false
				}
			} else if mon.x > mon.moveToX {
				mon.x -= moveSpeedPerTick
				mon.direction = "left"
				if mon.x < mon.moveToX {
					mon.x = mon.moveToX
					mon.isMoving = false
				}
			}

			if mon.y < mon.moveToY {
				mon.y += moveSpeedPerTick
				if mon.y > mon.moveToY {
					mon.y = mon.moveToY
					mon.isMoving = false
				}
			} else if mon.y > mon.moveToY {
				mon.y -= moveSpeedPerTick
				if mon.y < mon.moveToY {
					mon.y = mon.moveToY
					mon.isMoving = false
				}
			}

			if mon.x == mon.moveToX && mon.y == mon.moveToY {
				mon.isMoving = false
			}
		}
	}
}

func (g *Game) spawnInitialMonsters() {
	spawnLayer := g.gameMap.getLayerByName("spawns")
	if spawnLayer == nil {
		log.Println("no spawn layer found in map")
		return
	}

	for _, obj := range spawnLayer.Objects {
		var kind string
		var hp int
		switch obj.Name {
		case "archer":
			kind = monsterKindArcher
			hp = 100
		case "skeleton":
			kind = monsterKindSkeleton
			hp = 200
		default:
			continue
		}
		g.monsters = append(g.monsters, &Monster{
			id:        len(g.monsters) + 1,
			kind:      kind,
			hp:        hp,
			x:         int(obj.X),
			y:         int(obj.Y),
			direction: "left",
			isMoving:  false,
		})
	}
}

func (g *Game) spawnInitialObjects() {
	spawnLayer := g.gameMap.getLayerByName("objects")
	if spawnLayer == nil {
		log.Println("no objects layer found in map")
		return
	}

	for _, obj := range spawnLayer.Objects {
		var kind string
		var state string
		switch obj.Type {
		case "chest":
			kind = objectKindChest
			state = "closed"
		case "trigger":
			kind = objectKindTrigger
			state = "ready"
		case "trap_arrow":
			kind = objectKindTrapArrow
			state = "ready"
		case "trap_spikes":
			kind = objectKindTrapSpikes
			state = "ready"
		default:
			continue
		}

		propsMap := make(map[string]interface{})
		for _, prop := range obj.Properties {
			propsMap[prop.Name] = prop.Value
		}

		g.objects[uint64(len(g.objects)+1)] = &Object{
			ID:            len(g.objects) + 1,
			Kind:          kind,
			X:             int(obj.X),
			Y:             int(obj.Y),
			Width:         int(obj.Width),
			Height:        int(obj.Height),
			State:         state,
			PropertiesMap: propsMap,
		}
	}
}

func (g *Game) spawnDemonUnsafe() {
	spawnLayer := g.gameMap.getLayerByName("spawns")
	if spawnLayer == nil {
		log.Println("no spawn layer found in map")
		return
	}

	for _, obj := range spawnLayer.Objects {
		if obj.Name == "demon" {
			g.monsters = append(g.monsters, &Monster{
				id:        len(g.monsters) + 1,
				kind:      monsterKindDemon,
				hp:        1000,
				x:         int(obj.X),
				y:         int(obj.Y),
				direction: "left",
				isMoving:  false,
			})
			break
		}
	}

	g.demonWasSpawned = true
}

func getVectorFromDirection(direction string) (float64, float64) {
	switch direction {
	case "up":
		return 0, -1
	case "down":
		return 0, 1
	case "left":
		return -1, 0
	case "right":
		return 1, 0
	default:
		return 0, 0
	}
}

func (g *Game) isSwordAttackHit(attackerX, attackerY, attackLineX, attackLineY, targetX, targetY, targetRadius int) bool {
	return lineIntersectsRect(attackerX, attackerY, attackLineX, attackLineY, targetX-targetRadius, targetY-targetRadius, 2*targetRadius, 2*targetRadius) &&
		g.isVisible(attackerX, attackerY, targetX, targetY)
}
