package game

type CastEvent struct {
	SpellId        string `json:"spellId"`
	OriginPlayerId uint64 `json:originPlayerId`
}

type DamageEvent struct {
	SpellId        string `json:"spellId"`
	Damage         int    `json:"damage"`
	TargetPlayerId uint64 `json:"targetPlayerId"`
	TargetPlayerHp int    `json:"targetPlayerHp"`
	ShieldWorked   bool   `json:"shieldWorked"`
}

type PlayerPosition struct {
	ClientID  uint64 `json:"clientId"`
	Nickname  string `json:"nickname"`
	X         int    `json:"x"`
	Y         int    `json:"y"`
	Direction string `json:"direction"`
	IsMoving  bool   `json:"isMoving"`
}

type PlayerPositionsUpdateEvent struct {
	Players []PlayerPosition `json:"players"`
}

type EndGameEvent struct {
	WinnerPlayerId uint64 `json:"winnerPlayerId"`
}

type JoinToStartedGameEvent struct {
}
