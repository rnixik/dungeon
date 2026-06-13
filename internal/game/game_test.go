package game

import (
	"testing"
	"time"
)

func TestHitPlayerUnsafeReducesHP(t *testing.T) {
	g, broadcast := newTestGame()
	p, _ := addTestPlayer(g, 1, ClassKnight) // 250 HP

	g.hitPlayerUnsafe(1, 40)

	if p.hp != 210 {
		t.Errorf("hp = %d, want 210", p.hp)
	}
	if len(*broadcast) != 1 {
		t.Fatalf("expected 1 damage event, got %d", len(*broadcast))
	}
	if _, ok := (*broadcast)[0].(DamageEvent); !ok {
		t.Errorf("expected DamageEvent, got %T", (*broadcast)[0])
	}
}

func TestHitPlayerUnsafeClampsAndKills(t *testing.T) {
	g, broadcast := newTestGame()
	p, _ := addTestPlayer(g, 1, ClassMage) // 150 HP

	g.hitPlayerUnsafe(1, 999)

	if p.hp != 0 {
		t.Errorf("hp = %d, want 0 (clamped)", p.hp)
	}
	// Expect a DamageEvent followed by a PlayerDeathEvent.
	var sawDeath bool
	for _, e := range *broadcast {
		if _, ok := e.(PlayerDeathEvent); ok {
			sawDeath = true
		}
	}
	if !sawDeath {
		t.Error("expected a PlayerDeathEvent when HP reaches 0")
	}
}

func TestHitPlayerUnsafeIgnoresDeadPlayer(t *testing.T) {
	g, broadcast := newTestGame()
	p, _ := addTestPlayer(g, 1, ClassMage)
	p.hp = 0

	g.hitPlayerUnsafe(1, 10)

	if len(*broadcast) != 0 {
		t.Errorf("expected no events for already-dead player, got %d", len(*broadcast))
	}
}

func TestHitPlayerWithKindAppliesResistance(t *testing.T) {
	g, _ := newTestGame()
	// Mage takes half damage from fireball (40 -> 20).
	p, _ := addTestPlayer(g, 1, ClassMage)

	g.hitPlayerWithKindUnsafe(1, damageKindFireball)

	if p.hp != 150-20 {
		t.Errorf("hp = %d, want %d (fireball halved for mage)", p.hp, 150-20)
	}
}

func TestHitPlayerWithKindNoResistance(t *testing.T) {
	g, _ := newTestGame()
	// Mage has no resistance to arrows (full 30 damage).
	p, _ := addTestPlayer(g, 1, ClassMage)

	g.hitPlayerWithKindUnsafe(1, damageKindArrow)

	if p.hp != 150-30 {
		t.Errorf("hp = %d, want %d (arrow full damage)", p.hp, 150-30)
	}
}

func TestHitPlayerWithKindProtectionHalvesDamage(t *testing.T) {
	g, _ := newTestGame()
	p, _ := addTestPlayer(g, 1, ClassMage)
	p.protectionActiveUntil = time.Now().Add(time.Minute)

	// Arrow = 30, no class resistance, protection halves -> 15.
	g.hitPlayerWithKindUnsafe(1, damageKindArrow)

	if p.hp != 150-15 {
		t.Errorf("hp = %d, want %d (protection halves arrow)", p.hp, 150-15)
	}
}

func TestHitPlayerWithKindProtectionStacksWithResistance(t *testing.T) {
	g, _ := newTestGame()
	p, _ := addTestPlayer(g, 1, ClassMage)
	p.protectionActiveUntil = time.Now().Add(time.Minute)

	// Fireball 40 -> mage resistance 0.5 -> 20 -> protection /2 -> 10.
	g.hitPlayerWithKindUnsafe(1, damageKindFireball)

	if p.hp != 150-10 {
		t.Errorf("hp = %d, want %d (resistance then protection)", p.hp, 150-10)
	}
}

func TestAddXPLevelsUp(t *testing.T) {
	g, _ := newTestGame()
	p, _ := addTestPlayer(g, 1, ClassKnight) // 250 HP, nextLevelXP 500

	g.addXPToPlayerUnSafe(1, 500)

	if p.level != 2 {
		t.Errorf("level = %d, want 2", p.level)
	}
	if p.maxHp != 280 {
		t.Errorf("maxHp = %d, want 280 (+30 on level up)", p.maxHp)
	}
	if p.hp != 280 {
		t.Errorf("hp = %d, want 280 (full heal on level up)", p.hp)
	}
	// Leftover XP carries over (500 - 500 = 0) and threshold grows by 50%.
	if p.xp != 0 {
		t.Errorf("xp = %d, want 0 carried over", p.xp)
	}
	if p.nextLevelXP != 750 {
		t.Errorf("nextLevelXP = %d, want 750", p.nextLevelXP)
	}
}

func TestAddXPCapsAtLevelThree(t *testing.T) {
	g, _ := newTestGame()
	p, _ := addTestPlayer(g, 1, ClassRogue) // nextLevelXP 500

	// One level-up is granted per call; climb to the cap step by step.
	g.addXPToPlayerUnSafe(1, 500) // level 1 -> 2 (nextLevelXP becomes 750)
	g.addXPToPlayerUnSafe(1, 750) // level 2 -> 3 (capped)

	if p.level != 3 {
		t.Errorf("level = %d, want capped at 3", p.level)
	}
	// At max level, XP is pinned to the threshold (no overflow shown).
	if p.xp != p.nextLevelXP {
		t.Errorf("xp = %d, want pinned to nextLevelXP %d", p.xp, p.nextLevelXP)
	}

	prevHP := p.maxHp
	g.addXPToPlayerUnSafe(1, 100000) // further XP must not grant another level
	if p.level != 3 || p.maxHp != prevHP {
		t.Errorf("extra XP past level 3 changed state: level=%d maxHp=%d", p.level, p.maxHp)
	}
}

func TestHitMonsterUnsafeReducesHPAndAwardsKillXP(t *testing.T) {
	g, _ := newTestGame()
	_, client := addTestPlayer(g, 1, ClassKnight)
	mon := &Monster{id: 10, kind: monsterKindArcher, hp: 100, maxHP: 100}
	g.monsters = append(g.monsters, mon)

	g.hitMonsterUnsafe(1, 10, 30)
	if mon.hp != 70 {
		t.Errorf("monster hp = %d, want 70", mon.hp)
	}
	if mon.hitsTaken != 1 {
		t.Errorf("hitsTaken = %d, want 1", mon.hitsTaken)
	}

	// Kill it; player should be awarded kill XP via an XPEvent.
	g.hitMonsterUnsafe(1, 10, 999)
	if mon.hp != 0 {
		t.Errorf("monster hp = %d, want 0", mon.hp)
	}
	var sawXP bool
	for _, e := range client.sentEvents {
		if xp, ok := e.(XPEvent); ok && xp.XP >= xpPerMonsterKill {
			sawXP = true
		}
	}
	if !sawXP {
		t.Error("expected an XPEvent awarding kill XP")
	}
}

func TestHitMonsterUnsafeShieldReducesDamage(t *testing.T) {
	g, _ := newTestGame()
	addTestPlayer(g, 1, ClassKnight)
	mon := &Monster{id: 10, kind: monsterKindGolem, hp: 1000, maxHP: 1000,
		shieldUntil: time.Now().Add(time.Minute)}
	g.monsters = append(g.monsters, mon)

	// 100 damage shielded by 90% -> 10.
	g.hitMonsterUnsafe(1, 10, 100)
	if mon.hp != 990 {
		t.Errorf("monster hp = %d, want 990 (shielded)", mon.hp)
	}
}

func TestHitMonsterUnsafeIgnoresDeadOrMissing(t *testing.T) {
	g, _ := newTestGame()
	addTestPlayer(g, 1, ClassKnight)
	mon := &Monster{id: 10, kind: monsterKindArcher, hp: 0, maxHP: 100}
	g.monsters = append(g.monsters, mon)

	g.hitMonsterUnsafe(1, 10, 50) // already dead -> untouched
	if mon.hp != 0 {
		t.Errorf("dead monster hp = %d, want 0", mon.hp)
	}

	g.hitMonsterUnsafe(1, 999, 50) // missing monster -> no panic, no effect
}
