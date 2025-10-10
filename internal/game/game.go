package game

import (
	"dungeon/internal/lobby"
	"encoding/json"
	"log"
	"sync"
	"time"
)

const StatusStarted = "started"
const StatusEnded = "ended"

const maxHP = 1000
const fireballDamage = 25
const positionsUpdateTickPeriod = time.Second / 60
const commonUpdateTickPeriod = time.Second / 3

const monsterKindArcher = "archer"
const monsterKindSkeleton = "skeleton"
const monsterKindDemon = "demon"

const objectKindChest = "chest"

type Player struct {
	client             lobby.ClientPlayer
	lastSpellId        string
	lastSpellIdShield  string
	lastCastTime       time.Time
	lastCastTimeShield time.Time
	spellWasSent       bool
	spellWasSentShield bool
	hasActiveSpell     bool
	hp                 int
	x                  int
	y                  int
	direction          string
	isMoving           bool
}

type Monster struct {
	id              int
	kind            string
	hp              int
	x               int
	y               int
	direction       string
	isMoving        bool
	isAttacking     bool
	attacked        bool
	attackStartedAt time.Time
	moveToX         int
	moveToY         int
}

type Object struct {
	ID    int    `json:"id"`
	Kind  string `json:"kind"`
	X     int    `json:"x"`
	Y     int    `json:"y"`
	State string `json:"state"`
}

func newPlayer(client lobby.ClientPlayer) *Player {
	return &Player{
		client:         client,
		lastSpellId:    "",
		lastCastTime:   time.Time{},
		hasActiveSpell: false,
		hp:             maxHP,
		x:              120,
		y:              140,
		direction:      "right",
		isMoving:       false,
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
		g.hitMonster(c.OriginClientID, c.MonsterID)
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
	client.SendEvent(JoinToStartedGameEvent{GameData: g.GetJoinClientData()})
}

func (g *Game) GetJoinClientData() map[string]interface{} {
	return map[string]interface{}{
		"mapData":     g.gameMap,
		"gameObjects": g.objects,
	}
}

func (g *Game) StartMainLoop() {
	g.spawnInitialMonsters()
	g.spawnInitialObjects()
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
					Nickname: pl.client.Nickname(),
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

	g.mutex.Lock()
	if p, ok := g.players[clientID]; ok {
		if p.hp <= 0 {
			isDead = true
		}
	}
	g.mutex.Unlock()

	if isDead {
		return
	}

	g.broadcastEventFunc(FireballEvent{
		ClientID:  clientID,
		X:         x,
		Y:         y,
		Direction: direction,
	})
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

func (g *Game) hitMonster(originClientID uint64, monsterID int) {
	g.mutex.Lock()
	defer g.mutex.Unlock()
	for _, m := range g.monsters {
		if m.id == monsterID && m.hp > 0 {
			m.hp -= fireballDamage
			if m.hp < 0 {
				m.hp = 0
			}

			g.broadcastEventFunc(DamageEvent{
				TargetMonsterID: monsterID,
				Damage:          fireballDamage,
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
		case "demon":
			kind = monsterKindDemon
			hp = 1000
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
		switch obj.Name {
		case "chest":
			kind = objectKindChest
			state = "closed"
		default:
			continue
		}
		g.objects[uint64(len(g.objects)+1)] = &Object{
			ID:    len(g.objects) + 1,
			Kind:  kind,
			X:     int(obj.X),
			Y:     int(obj.Y),
			State: state,
		}
	}
}
