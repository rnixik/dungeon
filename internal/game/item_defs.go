package game

import (
	"fmt"
	"time"
)

// ItemDef describes a usable inventory item. Adding a new item is a single entry
// in itemDefs plus its effect method.
type ItemDef struct {
	Kind        string
	ConsumesOne bool                         // decrement the inventory count on use
	Use         func(*Game, *Player, uint64) // effect; runs under the game mutex
}

const (
	itemHealingPotion       = "healing_potion"
	itemScrollOfXP          = "scroll_of_xp"
	itemScrollOfFootprints  = "scroll_of_footprints"
	itemBootsOfHaste        = "boots_of_haste"
	itemScrollOfProtection  = "scroll_of_protection"
	itemSpikes              = "spikes"
	itemCloakOfInvisibility = "cloak_of_invisibility"
)

var itemDefs = map[string]*ItemDef{
	itemHealingPotion:       {Kind: itemHealingPotion, ConsumesOne: true, Use: (*Game).useHealingPotion},
	itemScrollOfXP:          {Kind: itemScrollOfXP, ConsumesOne: true, Use: (*Game).useScrollOfXP},
	itemScrollOfFootprints:  {Kind: itemScrollOfFootprints, ConsumesOne: true, Use: (*Game).useScrollOfFootprints},
	itemBootsOfHaste:        {Kind: itemBootsOfHaste, ConsumesOne: true, Use: (*Game).useBootsOfHaste},
	itemScrollOfProtection:  {Kind: itemScrollOfProtection, ConsumesOne: true, Use: (*Game).useScrollOfProtection},
	itemSpikes:              {Kind: itemSpikes, ConsumesOne: true, Use: (*Game).useSpikes},
	itemCloakOfInvisibility: {Kind: itemCloakOfInvisibility, ConsumesOne: false, Use: (*Game).useCloakOfInvisibilityUnsafe},
}

func (g *Game) useHealingPotion(p *Player, clientID uint64) {
	oldHp := p.hp
	p.hp += 50
	if p.hp > p.maxHp {
		p.hp = p.maxHp
	}
	p.client.SendEvent(HealEvent{
		ClientID: clientID,
		Amount:   p.hp - oldHp,
		HP:       p.hp,
		MaxHP:    p.maxHp,
	})
}

func (g *Game) useScrollOfXP(p *Player, clientID uint64) {
	g.addXPToPlayerUnSafe(clientID, 500)
}

func (g *Game) useScrollOfFootprints(p *Player, clientID uint64) {
	p.footprintsActiveUntil = time.Now().Add(30 * time.Second)
	if len(g.positionSnapshots) > 0 {
		histPoints := make([]FootprintPoint, 0, len(g.positionSnapshots)*len(g.players))
		for _, snap := range g.positionSnapshots {
			histPoints = append(histPoints, snap.points...)
		}
		if len(histPoints) > 0 {
			p.client.SendEvent(FootprintsEvent{Points: histPoints})
		}
	}
	expireClientID := clientID
	time.AfterFunc(30*time.Second, func() {
		g.mutex.Lock()
		defer g.mutex.Unlock()
		pl, ok := g.players[expireClientID]
		if !ok || time.Now().Before(pl.footprintsActiveUntil) {
			return
		}
		pl.client.SendEvent(FootprintsExpiredEvent{})
	})
}

func (g *Game) useBootsOfHaste(p *Player, clientID uint64) {
	const maxSpeedBoost = 30
	if p.speedBoostPercent < maxSpeedBoost {
		p.speedBoostPercent += maxSpeedBoost
		if p.speedBoostPercent > maxSpeedBoost {
			p.speedBoostPercent = maxSpeedBoost
		}
	}
}

func (g *Game) useScrollOfProtection(p *Player, clientID uint64) {
	const protectionDuration = 60 * time.Second
	p.protectionActiveUntil = time.Now().Add(protectionDuration)
	p.client.SendEvent(ProtectionActiveEvent{Duration: int(protectionDuration.Milliseconds())})
	expireClientID := clientID
	time.AfterFunc(protectionDuration, func() {
		g.mutex.Lock()
		defer g.mutex.Unlock()
		pl, ok := g.players[expireClientID]
		if !ok || time.Now().Before(pl.protectionActiveUntil) {
			return
		}
		pl.client.SendEvent(ProtectionExpiredEvent{})
	})
}

func (g *Game) useSpikes(p *Player, clientID uint64) {
	trapID := fmt.Sprintf("item_spike_%d_%d", clientID, time.Now().UnixNano())
	tileX := (p.x / tileSize) * tileSize
	tileY := (p.y / tileSize) * tileSize
	trap := NewTrap(trapID, TrapTypeSpikes, TrapParams{
		ActivePercent:   30.0,
		CooldownPercent: 20.0,
		Damage:          18,
		X:               tileX,
		Y:               tileY,
	}, TrapActivator{
		Type:   ActivatorTimer,
		Period: 2.0,
	})
	g.traps[trapID] = trap
	g.broadcastEventFunc(TrapStateChangedEvent{
		TrapID: trapID,
		State:  trap.State,
		X:      tileX,
		Y:      tileY,
		Frame:  trap.GetCurrentFrame(),
	})
}
