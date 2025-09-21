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
