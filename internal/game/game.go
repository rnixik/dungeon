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
}

func NewGame(playersClients []lobby.ClientPlayer, room *lobby.Room, broadcastEventFunc func(event interface{})) *Game {
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
			log.Printf("cannot decode MoveCommand: %v\n", err)

			return
		}
		g.movePlayerTo(client.ID(), c.X, c.Y, c.Direction, c.IsMoving)
		break
	}
}

func (g *Game) OnClientRemoved(client lobby.ClientPlayer) {
	if g.isGameEnded() {
		return
	}
}

func (g *Game) OnClientJoined(client lobby.ClientPlayer) {
	log.Printf("client '%s' joined game\n", client.Nickname())
}

func (g *Game) StartMainLoop() {
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if g.isGameEnded() {
				return
			}

			g.mutex.Lock()
			p := make([]Position, 0, len(g.players))
			for _, pl := range g.players {
				p = append(p, Position{
					ClientID:  pl.client.ID(),
					X:         pl.x,
					Y:         pl.y,
					Direction: pl.direction,
					IsMoving:  pl.isMoving,
				})
			}
			g.broadcastEventFunc(PositionUpdateEvent{
				Positions: p,
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
