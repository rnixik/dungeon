package game

type DamageEvent struct {
	TargetPlayerId  uint64 `json:"targetPlayerId"`
	TargetMonsterID int    `json:"targetMonsterId"`
	Damage          int    `json:"damage"`
}

type PlayerPosition struct {
	ClientID  uint64 `json:"clientId"`
	Nickname  string `json:"nickname"`
	X         int    `json:"x"`
	Y         int    `json:"y"`
	Direction string `json:"direction"`
	IsMoving  bool   `json:"isMoving"`
	HP        int    `json:"hp"`
}

type MonsterPosition struct {
	ID        int    `json:"id"`
	Kind      string `json:"kind"`
	X         int    `json:"x"`
	Y         int    `json:"y"`
	Direction string `json:"direction"`
	IsMoving  bool   `json:"isMoving"`
	HP        int    `json:"hp"`
}

type FireballEvent struct {
	ClientID  uint64 `json:"clientId"`
	X         int    `json:"x"`
	Y         int    `json:"y"`
	Direction string `json:"direction"`
}

type PlayerPositionsUpdateEvent struct {
	Players  []PlayerPosition  `json:"players"`
	Monsters []MonsterPosition `json:"monsters"`
}

type EndGameEvent struct {
	WinnerPlayerId uint64 `json:"winnerPlayerId"`
}

type JoinToStartedGameEvent struct {
}

type PlayerDeathEvent struct {
	ClientID uint64 `json:"clientID"`
}

type MonsterDeathEvent struct {
	ID int `json:"id"`
}
