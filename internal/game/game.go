package game

import (
	"dungeon/internal/lobby"
	"encoding/json"
	"log"
	"math/rand"
	"strconv"
	"sync"
	"time"
)

const StatusStarted = "started"
const StatusEnded = "ended"

const ClassMage = "mage"
const ClassKnight = "knight"
const ClassRogue = "rogue"

var playerColors = []string{
	"0xe74c3c", // red
	"0x3498db", // blue
	"0x2ecc71", // green
	"0xf1c40f", // yellow
	"0x9b59b6", // purple
	"0xe67e22", // orange
	"0x1abc9c", // teal
	"0xff69b4", // pink
	"0xcd853f", // brown
	"0x00bcd4", // cyan
	"0xff5722", // deep orange
	"0x8bc34a", // light green
	"0x673ab7", // deep purple
	"0xff9800", // amber
	"0x03a9f4", // light blue
	"0xe91e63", // rose
	"0xf06292", // light pink
	"0x26c6da", // teal accent
	"0xd4e157", // lime
	"0xa1887f", // warm brown
}

// isValidPlayerColor reports whether the given color is part of the selectable
// palette. Used to validate a client-chosen color before applying it.
func isValidPlayerColor(color string) bool {
	for _, c := range playerColors {
		if c == color {
			return true
		}
	}

	return false
}

const positionsUpdateTickPeriod = time.Second / 60
const commonUpdateTickPeriod = time.Second / 3

const monsterKindArcher = "archer"
const monsterKindSkeleton = "skeleton"
const monsterKindDemon = "demon"
const monsterKindGolem = "golem"
const monsterKindSpider = "spider"
const monsterKindJelly = "jelly"
const monsterKindJellySmall = "jelly_small"
const monsterKindJellyMicro = "jelly_micro"
const monsterKindDemonMage = "demon_mage"

const attackFireballCooldown = time.Second
const attackShotArrowCooldown = time.Second / 4
const attackSwordCooldown = time.Second
const attackSwordDelay = time.Millisecond * 700

const objectKindChest = "chest"
const objectKindTrigger = "trigger"
const objectKindTrapArrow = "trap_arrow"
const objectKindTrapSpikes = "trap_spikes"

const damageKindFireball = "fireball"
const damageKindArrow = "arrow"
const damageKindExplosion = "explosion"
const damageKindBullet = "bullet"
const damageKindFirespot = "firespot"
const damageKindSpike = "spike"
const damageKindLightning = "lightning"

const xpPerMonsterKill = 250

// cultistCurseChance is the probability that opening a chest curses the opener
// into becoming a cultist (the opposite team).
const cultistCurseChance = 0.3

// cultistMaxFraction caps the number of cultists at one third of the players
// (one cultist per two good players).
const cultistMaxFraction = 3

type Player struct {
	client                lobby.ClientPlayer
	class                 string
	avatarUrl             string
	lastAttackTime        time.Time
	color                 string
	level                 int
	xp                    int
	nextLevelXP           int
	maxHp                 int
	hp                    int
	x                     int
	y                     int
	direction             string
	isMoving              bool
	isDodging             bool
	inventory             []InventoryItem
	footprintsActiveUntil time.Time
	protectionActiveUntil time.Time
	invisibleUntil        time.Time
	cloakLastUsed         time.Time
	speedBoostPercent     int
	// Curse / cultist state
	isCultist bool
	// isSpectator is set once a player can no longer play after the boss is
	// revealed (a good player who died, a cultist out of Soul Power, or anyone who
	// joins/rejoins during the boss phase). Spectators keep receiving all events
	// so they can watch the battlemap, but their gameplay commands are ignored and
	// they are not broadcast to others as active players.
	isSpectator bool
	// goodDeathsBeforeBoss counts this player's deaths that fed Soul Power while
	// good and before the boss phase. They are uncounted if the player is cursed
	// into a cultist.
	goodDeathsBeforeBoss int
}

func (p *Player) isInvisible() bool {
	return !p.invisibleUntil.IsZero() && time.Now().Before(p.invisibleUntil)
}

type Monster struct {
	id                   int
	kind                 string
	hp                   int
	maxHP                int
	damage               int
	hitsTaken            int
	x                    int
	y                    int
	direction            string
	isMoving             bool
	isAttacking          bool
	attacked             bool
	attackStartedAt      time.Time
	moveToX              int
	moveToY              int
	path                 []Point
	pathGoalTX           int
	pathGoalTY           int
	firecircleStartedAt  time.Time
	lightningStartedAt   time.Time
	webStartedAt         time.Time
	shieldUntil          time.Time
	speedBoostUntil      time.Time
	spellTargetID        int
	spellIsShield        bool
	shieldLastCastAt     time.Time
	speedBoostLastCastAt time.Time
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
	// HasKey marks a chest that yields one of the three mandatory keys when
	// opened. Loot lists the items the opener receives. Both are populated at
	// game start by distributeChestLootUnsafe.
	HasKey bool       `json:"-"`
	Loot   []LootItem `json:"-"`
}

// LootItem is a quantity of a given item kind placed inside a chest.
type LootItem struct {
	Kind  string
	Count int
}

func newPlayer(client lobby.ClientPlayer) *Player {
	colorHex := playerColors[rand.Intn(len(playerColors))]

	class := classList[rand.Intn(len(classList))]

	props := client.GetAdditionalProperties()
	if cls, ok := props["class"].(string); ok {
		class = cls
	}

	// A player may pick a custom color (unlocked via the paywall on the client).
	// Only accept it if it belongs to the known palette, otherwise keep the
	// random one assigned above.
	if c, ok := props["color"].(string); ok && isValidPlayerColor(c) {
		colorHex = c
	}

	avatarUrl := ""
	if av, ok := props["avatarUrl"].(string); ok && len(av) <= 512 {
		avatarUrl = av
	}

	currentMaxHP := classMaxHP(class)

	return &Player{
		client:      client,
		class:       class,
		avatarUrl:   avatarUrl,
		color:       colorHex,
		level:       1,
		nextLevelXP: 500,
		maxHp:       currentMaxHP,
		hp:          currentMaxHP,
		x:           120,
		y:           140,
		direction:   "right",
		isMoving:    false,
		inventory: []InventoryItem{
			{Kind: itemHealingPotion, Count: 3},
			{Kind: itemSpikes, Count: 3},
			{Kind: itemScrollOfFootprints, Count: 1},
			{Kind: itemScrollOfXP, Count: 1},
			{Kind: itemBootsOfHaste, Count: 1},
			{Kind: itemScrollOfProtection, Count: 1},
			{Kind: itemCloakOfInvisibility, Count: 1},
		},
	}
}

type positionSnapshot struct {
	t      time.Time
	points []FootprintPoint
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
	spikeEvents        []SpawnSpikeEvent
	updateTilesEvents  []UpdateTilesEvent
	traps              map[string]*Trap
	positionSnapshots  []positionSnapshot
	// soulPower is a running tally: +1 for every good player that dies before the
	// boss phase, -1 for every cultist that dies before the boss phase. Once the
	// boss is revealed it is no longer fed by deaths and is instead spent by
	// cultists to respawn (1 per respawn).
	soulPower int
	// debug lets good players see the Soul Power value (used for local/dev
	// environments). Cultists always see it.
	debug bool
}

func NewGame(playersClients []lobby.ClientPlayer, room *lobby.Room, broadcastEventFunc func(event interface{}), gameMap *Map, debug bool) *Game {
	spawnX, spawnY := gameMap.PlayerSpawn()
	players := make(map[uint64]*Player, len(playersClients))
	for _, client := range playersClients {
		p := newPlayer(client)
		p.x, p.y = spawnX, spawnY
		players[client.ID()] = p
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
		debug:              debug,
		objects:            make(map[uint64]*Object),
		keysCollected: map[string]bool{
			"1": false,
			"2": false,
			"3": false,
		},
		spikeEvents:       make([]SpawnSpikeEvent, 0),
		updateTilesEvents: make([]UpdateTilesEvent, 0),
		traps:             make(map[string]*Trap),
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
		g.attackWithSword(client.ID())
		break
	case "ShootArrowCommand":
		var c ShootArrowCommand
		if err := json.Unmarshal(eventDataJson, &c); err != nil {
			log.Printf("cannot decode ShootArrowCommand: %v\n", err)

			return
		}
		g.shootArrow(client.ID(), c.X, c.Y, c.Direction)
		break
	case "DodgeCommand":
		var c DodgeCommand
		if err := json.Unmarshal(eventDataJson, &c); err != nil {
			log.Printf("cannot decode DodgeCommand: %v\n", err)

			return
		}
		g.dodge(client.ID(), c.X, c.Y, c.Direction, true)
		break
	case "HitPlayerCommand":
		var c HitPlayerCommand
		if err := json.Unmarshal(eventDataJson, &c); err != nil {
			log.Printf("cannot decode HitPlayerCommand: %v\n", err)

			return
		}

		g.mutex.Lock()
		g.hitPlayerWithKindUnsafe(c.TargetClientID, c.Kind)
		g.mutex.Unlock()
		break
	case "HitMonsterCommand":
		var c HitMonsterCommand
		if err := json.Unmarshal(eventDataJson, &c); err != nil {
			log.Printf("cannot decode HitMonsterCommand: %v\n", err)

			return
		}
		damage := damageForKind(c.Kind)
		g.hitMonster(c.OriginClientID, c.MonsterID, damage)
		break
	case "RespawnCommand":
		g.respawnPlayer(client.ID())
		break
	case "UseItemCommand":
		var c UseItemCommand
		if err := json.Unmarshal(eventDataJson, &c); err != nil {
			log.Printf("cannot decode UseItemCommand: %v\n", err)
			return
		}
		g.useItem(client.ID(), c.Kind)
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
	p := newPlayer(client)
	p.x, p.y = g.playerSpawn()
	if g.demonWasSpawned {
		// Anyone joining or rejoining after the boss is revealed cannot play; they
		// join as a spectator who can watch the battlemap.
		p.isSpectator = true
		p.hp = 0
		log.Printf("client '%s' joins as spectator (boss revealed)\n", client.Nickname())
	}
	g.players[client.ID()] = p
	if g.demonWasSpawned {
		g.checkCultistsWinUnsafe()
	}
	g.mutex.Unlock()
	client.SendEvent(JoinToStartedGameEvent{GameData: g.getPlayerInitialGameData(p)})
}

func (g *Game) GetCommonInitialGameData() map[string]interface{} {
	return map[string]interface{}{}
}

func (g *Game) getPlayerInitialGameData(pl *Player) map[string]interface{} {
	// Convert traps to initial state data
	trapsData := make([]map[string]interface{}, 0, len(g.traps))
	for _, trap := range g.traps {
		trapsData = append(trapsData, map[string]interface{}{
			"trapId": trap.ID,
			"state":  trap.State,
			"x":      trap.Params.X,
			"y":      trap.Params.Y,
			"frame":  trap.GetCurrentFrame(),
		})
	}

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
				IsDodging: pl.isDodging,
			},
			Class:       pl.class,
			Nickname:    pl.client.Nickname(),
			AvatarUrl:   pl.avatarUrl,
			Color:       pl.color,
			Level:       pl.level,
			XP:          pl.xp,
			NextLevelXP: pl.nextLevelXP,
			MaxHP:       pl.maxHp,
			HP:          pl.hp,
		},
		"keysCollected":     g.keysCollected,
		"spikeEvents":       g.spikeEvents,
		"updateTilesEvents": g.updateTilesEvents,
		"traps":             trapsData,
		"inventory":         pl.inventory,
		"speedBoostPercent": pl.speedBoostPercent,
		"soulPower":         g.soulPower,
		"soulPowerVisible":  pl.isCultist || g.debug,
		"isCultist":         pl.isCultist,
		"isSpectator":       pl.isSpectator,
		"bossRevealed":      g.demonWasSpawned,
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
				if pl.isSpectator || !pl.isMoving {
					continue
				}
				p = append(p, PlayerPosition{
					ClientID:  pl.client.ID(),
					X:         pl.x,
					Y:         pl.y,
					Direction: pl.direction,
					IsMoving:  pl.isMoving,
					IsDodging: pl.isDodging,
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

			g.mutex.Unlock()

			// Broadcast outside the lock (network sends shouldn't block command
			// handling), and skip entirely when nothing is moving or attacking.
			if len(p) > 0 || len(m) > 0 {
				g.broadcastEventFunc(CreaturesPosUpdateEvent{
					Players:  p,
					Monsters: m,
				})
			}
		case <-tickerCommon.C:
			if g.isGameEnded() {
				return
			}
			g.mutex.Lock()

			p := make([]PlayerStats, 0, len(g.players))
			for _, pl := range g.players {
				if pl.isSpectator {
					continue
				}
				p = append(p, PlayerStats{
					PlayerPosition: PlayerPosition{
						ClientID:  pl.client.ID(),
						X:         pl.x,
						Y:         pl.y,
						Direction: pl.direction,
						IsMoving:  pl.isMoving,
					},
					Class:             pl.class,
					Nickname:          pl.client.Nickname(),
					AvatarUrl:         pl.avatarUrl,
					Color:             pl.color,
					Level:             pl.level,
					MaxHP:             pl.maxHp,
					HP:                pl.hp,
					SpeedBoostPercent: pl.speedBoostPercent,
					HasShield:         !pl.protectionActiveUntil.IsZero() && time.Now().Before(pl.protectionActiveUntil),
					IsInvisible:       pl.isInvisible(),
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
					Kind:  mon.kind,
					HP:    mon.hp,
					MaxHP: mon.maxHP,
				})
			}

			now := time.Now()

			// Record position snapshot (all alive players including self)
			snap := positionSnapshot{
				t:      now,
				points: make([]FootprintPoint, 0, len(g.players)),
			}
			for _, pl := range g.players {
				if pl.hp > 0 {
					snap.points = append(snap.points, FootprintPoint{
						ClientID: pl.client.ID(),
						X:        pl.x,
						Y:        pl.y,
						Color:    pl.color,
					})
				}
			}
			cutoff := now.Add(-60 * time.Second)
			for len(g.positionSnapshots) > 0 && g.positionSnapshots[0].t.Before(cutoff) {
				g.positionSnapshots = g.positionSnapshots[1:]
			}
			g.positionSnapshots = append(g.positionSnapshots, snap)

			// Collect footprint recipients (players with an active scroll) while
			// holding the lock; the actual sends happen after unlocking.
			var footprintClients []lobby.ClientPlayer
			if len(snap.points) > 0 {
				for _, pl := range g.players {
					if pl.hp <= 0 || now.After(pl.footprintsActiveUntil) {
						continue
					}
					footprintClients = append(footprintClients, pl.client)
				}
			}

			g.mutex.Unlock()

			// Broadcast/send outside the lock so network sends don't block
			// command handling.
			g.broadcastEventFunc(CreaturesStatsUpdateEvent{
				Players:  p,
				Monsters: m,
			})
			for _, client := range footprintClients {
				client.SendEvent(FootprintsEvent{Points: snap.points})
			}
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
		if p.isSpectator {
			return
		}
		p.x = x
		p.y = y
		p.direction = direction
		p.isMoving = isMoving
	}
}

func (g *Game) dodge(clientID uint64, x int, y int, direction string, isMoving bool) {
	g.mutex.Lock()
	defer g.mutex.Unlock()
	if p, ok := g.players[clientID]; ok {
		if p.isSpectator {
			return
		}
		p.x = x
		p.y = y
		p.direction = direction
		p.isMoving = isMoving
		p.isDodging = true
	}
}

// beginAttack runs the shared preamble for a player attack: it reveals the
// attacker (clearing invisibility), aborts if the player is missing or dead,
// enforces the shared attack cooldown, and claims it. It returns the player to
// attack with, or nil if the attack should not proceed.
func (g *Game) beginAttack(clientID uint64, cooldown time.Duration) *Player {
	g.mutex.Lock()
	p, ok := g.players[clientID]
	dead := false
	if ok {
		dead = p.hp <= 0
		g.revealPlayerUnsafe(p)
	}
	g.mutex.Unlock()

	if !ok || dead {
		return nil
	}

	if time.Since(p.lastAttackTime) < cooldown {
		return nil
	}
	p.lastAttackTime = time.Now()

	return p
}

func (g *Game) castFireball(clientID uint64, x int, y int, direction string) {
	player := g.beginAttack(clientID, attackFireballCooldown)
	if player == nil {
		return
	}

	distance := 200 * player.level

	g.broadcastEventFunc(FireballEvent{
		ClientID:  clientID,
		X:         x,
		Y:         y,
		Direction: direction,
		Distance:  distance,
	})
}

func (g *Game) shootArrow(clientID uint64, x int, y int, direction string) {
	go func() {
		time.Sleep(time.Millisecond * 200)

		player := g.beginAttack(clientID, attackShotArrowCooldown)
		if player == nil {
			return
		}

		const dispersion = 100.0

		for i := 0; i < player.level; i++ {
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

			time.Sleep(time.Millisecond * 50)
		}
	}()
}

func (g *Game) attackWithSword(clientID uint64) {
	player := g.beginAttack(clientID, attackSwordCooldown)
	if player == nil {
		return
	}

	g.broadcastEventFunc(SwordAttackPrepareEvent{
		ClientID:  clientID,
		X:         player.x,
		Y:         player.y,
		Direction: player.direction,
	})

	go func() {
		time.Sleep(attackSwordDelay)

		isDead := false
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

		length := 50 + 60*player.level
		radius := 20 + 12*player.level
		damage := 50 + 5*(player.level-1)

		vecX, vecY := getVectorFromDirection(player.direction)
		attackX, attackY := player.x+int(vecX)*length, player.y+int(vecY)*length

		g.mutex.Lock()
		for _, p := range g.players {
			if p.client.ID() == clientID {
				continue
			}
			if (g.isSwordAttackHit(player.x, player.y, attackX, attackY, p.x, p.y, radius)) == false {
				continue
			}
			g.hitPlayerUnsafe(p.client.ID(), damage)
		}
		for _, m := range g.monsters {
			if (g.isSwordAttackHit(player.x, player.y, attackX, attackY, m.x, m.y, radius)) == false {
				continue
			}
			g.hitMonsterUnsafe(clientID, m.id, damage)
		}
		g.mutex.Unlock()

		g.broadcastEventFunc(SwordAttackEvent{
			ClientID:    clientID,
			X:           player.x,
			Y:           player.y,
			Direction:   player.direction,
			AttackLineX: attackX,
			AttackLineY: attackY,
			Radius:      radius,
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
			X:              p.x,
			Y:              p.y,
		})

		if p.hp == 0 {
			g.killPlayer(targetClientID)
		}
	}
}

func (g *Game) hitPlayerWithKindUnsafe(targetClientID uint64, kind string) {
	if p, ok := g.players[targetClientID]; ok {
		if p.hp == 0 {
			return
		}

		damage := damageForKind(kind)
		damage = int(float64(damage) * classResistance(p.class, kind))
		if !p.protectionActiveUntil.IsZero() && time.Now().Before(p.protectionActiveUntil) {
			damage = damage / 2
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
	p, ok := g.players[clientID]
	if !ok {
		return
	}

	wasAlive := p.hp > 0
	p.hp = 0

	g.broadcastEventFunc(PlayerDeathEvent{
		ClientID: clientID,
		Nickname: p.client.Nickname(),
	})

	if !wasAlive {
		return
	}

	// Soul Power accounting. Before the boss is revealed a cultist death drains
	// it and a good death feeds it. After the boss is revealed deaths no longer
	// move it via this path: cultists instead spend it to respawn (see
	// respawnPlayer), and good players are eliminated for good.
	if !g.demonWasSpawned {
		if p.isCultist {
			g.soulPower--
		} else {
			p.goodDeathsBeforeBoss++
			g.soulPower++
		}
		g.broadcastSoulPowerUnsafe()
	} else if !p.isCultist {
		// A good player who dies after the boss is revealed cannot play anymore.
		// They become a spectator who can still watch the battlemap.
		p.isSpectator = true
		g.checkCultistsWinUnsafe()
	}
}

func (g *Game) hitMonster(originClientID uint64, monsterID int, damage int) {
	g.mutex.Lock()
	defer g.mutex.Unlock()
	g.hitMonsterUnsafe(originClientID, monsterID, damage)
}

func (g *Game) hitMonsterUnsafe(originClientID uint64, monsterID int, damage int) {
	for _, m := range g.monsters {
		if m.id == monsterID && m.hp > 0 {
			if !m.shieldUntil.IsZero() && time.Now().Before(m.shieldUntil) {
				damage = damage / 10 // 90% reduction
				if damage < 1 {
					damage = 1
				}
			}
			m.hp -= damage
			if m.hp < 0 {
				m.hp = 0
			}
			m.hitsTaken++

			g.broadcastEventFunc(DamageEvent{
				TargetMonsterID: monsterID,
				Damage:          damage,
				X:               m.x,
				Y:               m.y,
			})

			if def := monsterDefs[m.kind]; def != nil && def.OnHit != nil {
				def.OnHit(g, m, originClientID)
			} else {
				g.defaultOnHit(m, originClientID)
			}

			// Destroying the demon cleanses the dungeon: the Light wins.
			if m.kind == monsterKindDemon && m.hp == 0 {
				g.endGame(originClientID, winningSideLight)
			}

			break
		}
	}
}

func (g *Game) splitJellyUnsafe(mon *Monster, originClientID uint64) {
	miniHP := mon.hp / 2
	if miniHP < 1 {
		miniHP = 1
	}
	miniDamage := mon.damage / 2
	if miniDamage < 5 {
		miniDamage = 5
	}

	mon.hp = 0
	g.addXPToPlayerUnSafe(originClientID, xpPerMonsterKill)

	g.broadcastEventFunc(JellySplitEvent{
		MonsterID: mon.id,
		X:         mon.x,
		Y:         mon.y,
	})

	childKind := monsterKindJellySmall
	if mon.kind == monsterKindJellySmall {
		childKind = monsterKindJellyMicro
	}

	// Delay spawning until split animation completes on client (13 frames @ 8fps ≈ 1625ms)
	spawnX := mon.x
	spawnY := mon.y
	time.AfterFunc(1700*time.Millisecond, func() {
		if g.isGameEnded() {
			return
		}
		g.mutex.Lock()
		defer g.mutex.Unlock()
		offsets := []int{-tileSize, tileSize}
		for _, offsetX := range offsets {
			g.monsters = append(g.monsters, &Monster{
				id:        len(g.monsters) + 1,
				kind:      childKind,
				hp:        miniHP,
				maxHP:     miniHP,
				damage:    miniDamage,
				x:         spawnX + offsetX,
				y:         spawnY,
				direction: "left",
			})
		}
	})
}

func (g *Game) addXPToPlayerUnSafe(clientID uint64, xp int) {
	if p, ok := g.players[clientID]; ok {
		p.xp += xp
		gotNewLevel := false
		if p.xp >= p.nextLevelXP && p.level < 3 {
			p.maxHp += 30
			p.hp = p.maxHp
			p.level = p.level + 1
			gotNewLevel = true

			if p.level == 3 {
				p.xp = p.nextLevelXP
			} else {
				p.xp -= p.nextLevelXP
				p.nextLevelXP += p.nextLevelXP / 2
			}
		}

		p.client.SendEvent(XPEvent{
			TargetPlayerId: clientID,
			XP:             p.xp,
			NextLevelXP:    p.nextLevelXP,
			Level:          p.level,
			GotNewLevel:    gotNewLevel,
		})
	}
}

func (g *Game) endGame(winnerPlayerId uint64, winningSide string) {
	g.statusMx.Lock()
	if g.status == StatusEnded {
		g.statusMx.Unlock()
		return
	}
	g.status = StatusEnded
	g.statusMx.Unlock()

	roles := make([]PlayerRole, 0, len(g.players))
	for _, p := range g.players {
		roles = append(roles, PlayerRole{
			ClientID:  p.client.ID(),
			Nickname:  p.client.Nickname(),
			Color:     p.color,
			IsCultist: p.isCultist,
		})
	}

	g.broadcastEventFunc(EndGameEvent{
		WinnerPlayerId: winnerPlayerId,
		WinningSide:    winningSide,
		Roles:          roles,
	})
	if g.room != nil {
		g.room.OnGameEnded()
	}
}

const (
	winningSideLight    = "light"
	winningSideCultists = "cultists"
)

// checkCultistsWinUnsafe ends the game in the cultists' favour once the boss is
// revealed and no good player is left fighting. Called from the good-player
// elimination paths, so it only fires after good players have actually fallen.
func (g *Game) checkCultistsWinUnsafe() {
	if !g.demonWasSpawned || g.isGameEnded() {
		return
	}
	for _, p := range g.players {
		if !p.isCultist && !p.isSpectator && p.hp > 0 {
			return // a good player is still in the fight
		}
	}
	g.endGame(0, winningSideCultists)
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

		moveSpeedPerTick := defaultMonsterMoveSpeed
		if def := monsterDefs[mon.kind]; def != nil {
			moveSpeedPerTick = def.MoveSpeed
		}
		if !mon.speedBoostUntil.IsZero() && time.Now().Before(mon.speedBoostUntil) {
			moveSpeedPerTick = (moveSpeedPerTick*3 + 1) / 2 // 50% boost
		}

		if mon.isMoving {
			newX := mon.x
			newY := mon.y
			newDir := mon.direction

			if mon.x < mon.moveToX {
				newX += moveSpeedPerTick
				newDir = "right"
				if newX > mon.moveToX {
					newX = mon.moveToX
				}
			} else if mon.x > mon.moveToX {
				newX -= moveSpeedPerTick
				newDir = "left"
				if newX < mon.moveToX {
					newX = mon.moveToX
				}
			}

			if mon.y < mon.moveToY {
				newY += moveSpeedPerTick
				if newY > mon.moveToY {
					newY = mon.moveToY
				}
			} else if mon.y > mon.moveToY {
				newY -= moveSpeedPerTick
				if newY < mon.moveToY {
					newY = mon.moveToY
				}
			}

			mon.x = newX
			mon.y = newY
			mon.direction = newDir

			if mon.x == mon.moveToX && mon.y == mon.moveToY {
				if len(mon.path) > 0 {
					mon.path = mon.path[1:]
				}
				if len(mon.path) > 0 {
					mon.moveToX = mon.path[0].X
					mon.moveToY = mon.path[0].Y
				} else {
					mon.isMoving = false
				}
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
		def := monsterDefBySpawnName(obj.Name)
		if def == nil || !def.SpawnOnStart {
			continue
		}
		g.monsters = append(g.monsters, &Monster{
			id:        len(g.monsters) + 1,
			kind:      def.Kind,
			hp:        def.BaseHP,
			maxHP:     def.BaseHP,
			damage:    def.Damage,
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

		// Parse properties first for all object types
		propsMap := make(map[string]interface{})
		for _, prop := range obj.Properties {
			propsMap[prop.Name] = prop.Value
		}

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

			// Create new trap instance using FSM
			tileX := (int(obj.X) / tileSize) * tileSize
			tileY := (int(obj.Y) / tileSize) * tileSize

			trapID := obj.Name
			if trapID == "" {
				trapID = "trap_" + string(rune(len(g.traps)+1))
			}

			// Default trap parameters (percent-based timing)
			// Default: 70% armed, 10% active (with rising animation), 20% cooldown
			params := TrapParams{
				ActivePercent:   10.0,
				CooldownPercent: 20.0,
				Damage:          18,
				X:               tileX,
				Y:               tileY,
			}

			// Override from properties if provided
			if activePercent, ok := propsMap["activePercent"].(float64); ok {
				params.ActivePercent = activePercent
			}
			if cooldownPercent, ok := propsMap["cooldownPercent"].(float64); ok {
				params.CooldownPercent = cooldownPercent
			}
			if damage, ok := propsMap["damage"].(float64); ok {
				params.Damage = int(damage)
			}

			// Validate and normalize percentages (total should not exceed 100%)
			totalPercent := params.ActivePercent + params.CooldownPercent
			if totalPercent > 100 {
				// Scale down to fit 100%
				scale := 100.0 / totalPercent
				params.ActivePercent *= scale
				params.CooldownPercent *= scale
			}
			// Armed percent is implicit: 100% - active% - cooldown%

			// Parse activator from properties
			activator := TrapActivator{
				Type:   ActivatorTimer,
				Period: 4.0, // Default 4 second period
				Phase:  0,
			}

			if activatorType, ok := propsMap["activator"].(string); ok {
				switch activatorType {
				case "timer":
					activator.Type = ActivatorTimer
					if period, ok := propsMap["period"].(float64); ok {
						activator.Period = period
					}
					// Parse phase - support both float and string
					if phase, ok := propsMap["phase"].(float64); ok {
						activator.Phase = phase
					} else if phaseStr, ok := propsMap["phase"].(string); ok {
						if phaseVal, err := strconv.ParseFloat(phaseStr, 64); err == nil {
							activator.Phase = phaseVal
						}
					}
				case "link":
					activator.Type = ActivatorLink
					if linkID, ok := propsMap["linkId"].(string); ok {
						activator.LinkID = linkID
					}
				}
			}

			trap := NewTrap(trapID, TrapTypeSpikes, params, activator)
			g.traps[trapID] = trap

			// Store trapId in object properties for linking
			propsMap["trapId"] = trapID

		default:
			continue
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

	g.distributeChestLootUnsafe()
}

// chestLootSpec defines the inclusive count range placed into a single chest for
// an optional item kind. Items with min == max always yield that fixed amount.
type chestLootSpec struct {
	kind     string
	minCount int
	maxCount int
}

var chestLootSpecs = []chestLootSpec{
	{itemHealingPotion, 1, 5},
	{itemScrollOfXP, 1, 2},
	{itemScrollOfFootprints, 1, 1},
	{itemBootsOfHaste, 1, 1},
	{itemScrollOfProtection, 1, 3},
	{itemSpikes, 3, 15},
	{itemCloakOfInvisibility, 1, 1},
}

// distributeChestLootUnsafe spreads the game's loot across all chests on the map
// at game start. The three keys are mandatory; every other item kind is optional
// and placed into a separate chest where possible. Caller must hold g.mutex (or
// run during single-goroutine setup).
func (g *Game) distributeChestLootUnsafe() {
	chests := make([]*Object, 0, len(g.objects))
	for _, obj := range g.objects {
		if obj.Kind == objectKindChest {
			chests = append(chests, obj)
		}
	}
	if len(chests) == 0 {
		return
	}

	rand.Shuffle(len(chests), func(i, j int) {
		chests[i], chests[j] = chests[j], chests[i]
	})

	// Hand out loot to distinct chests in shuffled order, wrapping around only if
	// there are fewer chests than loot bundles.
	next := 0
	assign := func(fn func(c *Object)) {
		fn(chests[next%len(chests)])
		next++
	}

	// Three keys are mandatory.
	for i := 0; i < 3; i++ {
		assign(func(c *Object) { c.HasKey = true })
	}

	// Optional items, one kind per chest.
	for _, spec := range chestLootSpecs {
		count := spec.minCount
		if spec.maxCount > spec.minCount {
			count += rand.Intn(spec.maxCount - spec.minCount + 1)
		}
		assign(func(c *Object) {
			c.Loot = append(c.Loot, LootItem{Kind: spec.kind, Count: count})
		})
	}
}

func (g *Game) spawnDemonUnsafe() {
	spawnLayer := g.gameMap.getLayerByName("spawns")
	if spawnLayer == nil {
		log.Println("no spawn layer found in map")
		return
	}

	def := monsterDefs[monsterKindDemon]
	for _, obj := range spawnLayer.Objects {
		if obj.Name == def.SpawnName {
			g.monsters = append(g.monsters, &Monster{
				id:        len(g.monsters) + 1,
				kind:      def.Kind,
				hp:        def.BaseHP,
				x:         int(obj.X),
				y:         int(obj.Y),
				direction: "left",
				isMoving:  false,
			})
			break
		}
	}

	g.demonWasSpawned = true
	g.moveLivingPlayersToBossStageUnsafe()
	g.broadcastEventFunc(BossRevealedEvent{})
}

// moveLivingPlayersToBossStageUnsafe teleports every living player into the boss
// arena when the boss phase begins, so the fight takes place on the boss stage.
// It is a no-op for maps that do not define a spawn_boss point.
func (g *Game) moveLivingPlayersToBossStageUnsafe() {
	x, y, ok := g.gameMap.BossSpawn()
	if !ok {
		return
	}

	for clientID, p := range g.players {
		if p.hp <= 0 {
			continue
		}
		p.x, p.y = x, y
		p.isMoving = false
		g.broadcastEventFunc(PlayerTeleportEvent{
			ClientID: clientID,
			X:        x,
			Y:        y,
		})
	}
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

// playerSpawn returns the spawn position for the current phase: the boss spawn
// once the boss has been revealed (when the map defines a spawn_boss), otherwise
// the normal start spawn.
func (g *Game) playerSpawn() (int, int) {
	if g.demonWasSpawned {
		if x, y, ok := g.gameMap.BossSpawn(); ok {
			return x, y
		}
	}
	return g.gameMap.PlayerSpawn()
}

func (g *Game) respawnPlayer(clientID uint64) {
	g.mutex.Lock()
	defer g.mutex.Unlock()
	p, ok := g.players[clientID]
	if !ok || p.hp > 0 {
		return
	}

	// Before the boss is revealed respawns are free for everyone. After it is
	// revealed good players are eliminated, while cultists may respawn only while
	// Soul Power remains, spending one point per respawn.
	if g.demonWasSpawned {
		if !p.isCultist {
			// Good players are out of the fight once the boss is revealed.
			p.isSpectator = true
			p.client.SendEvent(RespawnDeniedEvent{Reason: respawnDeniedEliminated})
			g.checkCultistsWinUnsafe()
			return
		}
		if g.soulPower <= 0 {
			// A cultist with no Soul Power left can no longer respawn.
			p.isSpectator = true
			p.client.SendEvent(RespawnDeniedEvent{Reason: respawnDeniedNoSoulPower})
			return
		}
		g.soulPower--
		g.broadcastSoulPowerUnsafe()
	}

	p.hp = p.maxHp
	p.x, p.y = g.playerSpawn()
	p.direction = "right"
	p.isMoving = false

	g.broadcastEventFunc(PlayerRespawnEvent{
		ClientID: clientID,
		X:        p.x,
		Y:        p.y,
	})
}

// giveItemToPlayerUnsafe adds count of the given item kind to the player's
// inventory (stacking onto an existing entry) and pushes an inventory update.
// Unknown kinds and non-positive counts are ignored. Caller must hold g.mutex.
func (g *Game) giveItemToPlayerUnsafe(p *Player, kind string, count int) {
	if _, ok := itemDefs[kind]; !ok || count <= 0 {
		return
	}
	for i := range p.inventory {
		if p.inventory[i].Kind == kind {
			p.inventory[i].Count += count
			g.sendInventoryUpdateUnsafe(p)
			return
		}
	}
	p.inventory = append(p.inventory, InventoryItem{Kind: kind, Count: count})
	g.sendInventoryUpdateUnsafe(p)
}

func (g *Game) sendInventoryUpdateUnsafe(p *Player) {
	const cloakCooldown = 3 * time.Minute
	items := make([]InventoryItem, len(p.inventory))
	copy(items, p.inventory)
	for i, item := range items {
		if item.Kind == itemCloakOfInvisibility && !p.cloakLastUsed.IsZero() {
			remaining := time.Until(p.cloakLastUsed.Add(cloakCooldown))
			if remaining > 0 {
				items[i].CooldownMs = int(remaining.Milliseconds())
			}
		}
	}
	p.client.SendEvent(InventoryUpdateEvent{
		ClientID:  p.client.ID(),
		Inventory: items,
	})
}

func (g *Game) revealPlayerUnsafe(p *Player) {
	if !p.isInvisible() {
		return
	}
	p.invisibleUntil = time.Time{}
	p.client.SendEvent(CloakExpiredEvent{})
	g.sendInventoryUpdateUnsafe(p)
}

// broadcastSoulPowerUnsafe sends the current Soul Power tally to each client.
// Cultists always see the value; good players only see it when debug is enabled.
func (g *Game) broadcastSoulPowerUnsafe() {
	for _, p := range g.players {
		p.client.SendEvent(SoulPowerEvent{
			Value:   g.soulPower,
			Visible: p.isCultist || g.debug,
		})
	}
}

// cultistCountUnsafe returns the number of current cultists.
func (g *Game) cultistCountUnsafe() int {
	n := 0
	for _, p := range g.players {
		if p.isCultist {
			n++
		}
	}

	return n
}

// maxCultistsAllowedUnsafe caps cultists at a third of the players (one cultist
// per two good players).
func (g *Game) maxCultistsAllowedUnsafe() int {
	return len(g.players) / cultistMaxFraction
}

// broadcastCultistsRosterUnsafe sends the list of cultist client IDs to every
// cultist so they can recognise one another. Good players are never told.
func (g *Game) broadcastCultistsRosterUnsafe() {
	ids := make([]uint64, 0)
	for _, p := range g.players {
		if p.isCultist {
			ids = append(ids, p.client.ID())
		}
	}
	for _, p := range g.players {
		if p.isCultist {
			p.client.SendEvent(CultistsRosterEvent{ClientIDs: ids})
		}
	}
}

// makePlayerCultistUnsafe curses a player into a cultist. Soul Power is
// recalculated so this player's earlier good deaths no longer count.
func (g *Game) makePlayerCultistUnsafe(p *Player) {
	if p.isCultist {
		return
	}
	p.isCultist = true

	// Uncount the deaths this player fed into Soul Power while good.
	if p.goodDeathsBeforeBoss > 0 {
		g.soulPower -= p.goodDeathsBeforeBoss
		p.goodDeathsBeforeBoss = 0
	}

	p.client.SendEvent(BecameCultistEvent{})
	g.broadcastCultistsRosterUnsafe()
	g.broadcastSoulPowerUnsafe()
	if g.demonWasSpawned {
		g.checkCultistsWinUnsafe()
	}
}

func (g *Game) useItem(clientID uint64, kind string) {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	p, ok := g.players[clientID]
	if !ok || p.hp <= 0 {
		return
	}

	def := itemDefs[kind]
	if def == nil {
		return
	}

	if def.ConsumesOne {
		idx := -1
		for i, item := range p.inventory {
			if item.Kind == kind && item.Count > 0 {
				idx = i
				break
			}
		}
		if idx == -1 {
			return
		}
		p.inventory[idx].Count--
	}

	def.Use(g, p, clientID)

	// Consumable items report their new count here; the cloak (non-consumable)
	// manages its own inventory updates because of its cooldown timers.
	if def.ConsumesOne {
		g.sendInventoryUpdateUnsafe(p)
	}
}

func (g *Game) useCloakOfInvisibilityUnsafe(p *Player, clientID uint64) {
	const cloakDuration = 30 * time.Second
	const cloakCooldown = 3 * time.Minute

	hasCloakItem := false
	for _, item := range p.inventory {
		if item.Kind == itemCloakOfInvisibility {
			hasCloakItem = true
			break
		}
	}
	if !hasCloakItem {
		return
	}

	if !p.cloakLastUsed.IsZero() && time.Now().Before(p.cloakLastUsed.Add(cloakCooldown)) {
		return
	}

	p.invisibleUntil = time.Now().Add(cloakDuration)
	p.cloakLastUsed = time.Now()

	p.client.SendEvent(CloakActiveEvent{
		Duration:   int(cloakDuration.Milliseconds()),
		CooldownMs: int(cloakCooldown.Milliseconds()),
	})
	g.sendInventoryUpdateUnsafe(p)

	time.AfterFunc(cloakDuration, func() {
		g.mutex.Lock()
		defer g.mutex.Unlock()
		pl, ok := g.players[clientID]
		if !ok || time.Now().Before(pl.invisibleUntil) {
			return
		}
		pl.invisibleUntil = time.Time{}
		pl.client.SendEvent(CloakExpiredEvent{})
		g.sendInventoryUpdateUnsafe(pl)
	})

	time.AfterFunc(cloakCooldown, func() {
		g.mutex.Lock()
		defer g.mutex.Unlock()
		pl, ok := g.players[clientID]
		if !ok {
			return
		}
		g.sendInventoryUpdateUnsafe(pl)
	})
}
