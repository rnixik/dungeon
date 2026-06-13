package game

// MonsterDef holds the static configuration and behavior for a monster kind.
// Adding a new monster is a single registerMonster call (plus its monsterKind*
// constant in game.go and an intellect* function in game_monsters.go).
type MonsterDef struct {
	Kind         string
	SpawnName    string // name in the Tiled "spawns" layer; "" if never map-spawned
	SpawnOnStart bool   // spawned by spawnInitialMonsters (demon and jelly children are not)
	BaseHP       int
	Damage       int                           // mon.damage at spawn (0 if unused)
	MoveSpeed    int                           // px per position-tick
	Intellect    func(*Game, *Monster)         // AI tick
	OnHit        func(*Game, *Monster, uint64) // post-damage hook; nil = defaultOnHit
}

const defaultMonsterMoveSpeed = 2

var monsterDefs = map[string]*MonsterDef{}

func registerMonster(d *MonsterDef) {
	if d.MoveSpeed == 0 {
		d.MoveSpeed = defaultMonsterMoveSpeed
	}
	monsterDefs[d.Kind] = d
}

// monsterDefBySpawnName returns the def whose SpawnName matches name, or nil.
func monsterDefBySpawnName(name string) *MonsterDef {
	for _, d := range monsterDefs {
		if d.SpawnName == name {
			return d
		}
	}

	return nil
}

// defaultOnHit awards kill XP when the monster dies. Used when a def has no OnHit.
func (g *Game) defaultOnHit(m *Monster, originClientID uint64) {
	if m.hp == 0 {
		g.addXPToPlayerUnSafe(originClientID, xpPerMonsterKill)
	}
}

// jellyOnHit splits the jelly once it has taken enough hits, otherwise behaves
// like defaultOnHit.
func (g *Game) jellyOnHit(m *Monster, originClientID uint64) {
	if m.hitsTaken >= 3 && m.hp > 0 {
		g.splitJellyUnsafe(m, originClientID)
	} else if m.hp == 0 {
		g.addXPToPlayerUnSafe(originClientID, xpPerMonsterKill)
	}
}

func init() {
	registerMonster(&MonsterDef{
		Kind:         monsterKindArcher,
		SpawnName:    "archer",
		SpawnOnStart: true,
		BaseHP:       100,
		Intellect:    (*Game).intellectArcher,
	})
	registerMonster(&MonsterDef{
		Kind:         monsterKindSkeleton,
		SpawnName:    "skeleton",
		SpawnOnStart: true,
		BaseHP:       200,
		Intellect:    (*Game).intellectSkeleton,
	})
	registerMonster(&MonsterDef{
		Kind:         monsterKindGolem,
		SpawnName:    "golem",
		SpawnOnStart: true,
		BaseHP:       1000,
		MoveSpeed:    1,
		Intellect:    (*Game).intellectGolem,
	})
	registerMonster(&MonsterDef{
		Kind:         monsterKindSpider,
		SpawnName:    "spider",
		SpawnOnStart: true,
		BaseHP:       150,
		Intellect:    (*Game).intellectSpider,
	})
	registerMonster(&MonsterDef{
		Kind:         monsterKindJelly,
		SpawnName:    "jelly",
		SpawnOnStart: true,
		BaseHP:       500,
		Damage:       20,
		MoveSpeed:    1,
		Intellect:    (*Game).intellectJelly,
		OnHit:        (*Game).jellyOnHit,
	})
	registerMonster(&MonsterDef{
		Kind:         monsterKindDemonMage,
		SpawnName:    "demon_mage",
		SpawnOnStart: true,
		BaseHP:       300,
		Intellect:    (*Game).intellectDemonMage,
	})
	registerMonster(&MonsterDef{
		Kind:         monsterKindDemon,
		SpawnName:    "demon",
		SpawnOnStart: false, // spawned only when all keys are collected
		BaseHP:       1000,
		Intellect:    (*Game).intellectDemon,
	})
	registerMonster(&MonsterDef{
		Kind:      monsterKindJellySmall,
		MoveSpeed: 1,
		Intellect: (*Game).intellectJelly,
		OnHit:     (*Game).jellyOnHit,
	})
	registerMonster(&MonsterDef{
		Kind:      monsterKindJellyMicro,
		MoveSpeed: 1,
		Intellect: (*Game).intellectJelly,
	})
}
