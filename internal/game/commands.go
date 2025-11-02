package game

type DemoCommand struct {
	DemoMessage string `json:"demoMessage"`
}

type CastCommand struct {
	SpellId string `json:"spellId"`
}

type MoveCommand struct {
	X         int    `json:"x"`
	Y         int    `json:"y"`
	Direction string `json:"direction"`
	IsMoving  bool   `json:"isMoving"`
}

type CastFireballCommand struct {
	X         int    `json:"x"`
	Y         int    `json:"y"`
	Direction string `json:"direction"`
}

type SwordAttackCommand struct {
	X         int    `json:"x"`
	Y         int    `json:"y"`
	Direction string `json:"direction"`
}

type ShootArrowCommand struct {
	X         int    `json:"x"`
	Y         int    `json:"y"`
	Direction string `json:"direction"`
}

type HitPlayerCommand struct {
	OriginClientID uint64 `json:"originClientId"`
	MonsterID      int    `json:"monsterId"`
	TargetClientID uint64 `json:"targetClientId"`
}

type HitMonsterCommand struct {
	OriginClientID uint64 `json:"originClientId"`
	MonsterID      int    `json:"monsterId"`
}
