package game

type DamageEvent struct {
	TargetPlayerId  uint64 `json:"targetPlayerId"`
	TargetMonsterID int    `json:"targetMonsterId"`
	Damage          int    `json:"damage"`
}

type FireballEvent struct {
	ClientID  uint64 `json:"clientId"`
	X         int    `json:"x"`
	Y         int    `json:"y"`
	Direction string `json:"direction"`
}

type ArrowEvent struct {
	ClientID  uint64 `json:"clientId"`
	MonsterID int    `json:"monsterId"`
	X1        int    `json:"x1"`
	Y1        int    `json:"y1"`
	X2        int    `json:"x2"`
	Y2        int    `json:"y2"`
}

type DemonFireballEvent struct {
	ClientID  uint64 `json:"clientId"`
	MonsterID int    `json:"monsterId"`
	X1        int    `json:"x1"`
	Y1        int    `json:"y1"`
	X2        int    `json:"x2"`
	Y2        int    `json:"y2"`
}

type FireCircleEvent struct {
	ClientID  uint64 `json:"clientId"`
	MonsterID int    `json:"monsterId"`
	X         int    `json:"x"`
	Y         int    `json:"y"`
}

type DemonLightningEvent struct {
	MonsterID int `json:"monsterId"`
	X         int `json:"x"`
	Y         int `json:"y"`
}

type PlayerPosition struct {
	ClientID  uint64 `json:"clientId"`
	X         int    `json:"x"`
	Y         int    `json:"y"`
	Direction string `json:"direction"`
	IsMoving  bool   `json:"isMoving"`
}

type MonsterPosition struct {
	ID          int    `json:"id"`
	X           int    `json:"x"`
	Y           int    `json:"y"`
	Direction   string `json:"direction"`
	IsMoving    bool   `json:"isMoving"`
	IsAttacking bool   `json:"isAttacking"`
}

type CreaturesPosUpdateEvent struct {
	Players  []PlayerPosition  `json:"players"`
	Monsters []MonsterPosition `json:"monsters"`
}

type EndGameEvent struct {
	WinnerPlayerId uint64 `json:"winnerPlayerId"`
}

type PlayerStats struct {
	PlayerPosition
	Nickname string `json:"nickname"`
	Color    string `json:"color"`
	MaxHP    int    `json:"maxHp"`
	HP       int    `json:"hp"`
}

type MonsterStats struct {
	MonsterPosition
	Kind string `json:"kind"`
	HP   int    `json:"hp"`
}

type CreaturesStatsUpdateEvent struct {
	Players  []PlayerStats  `json:"players"`
	Monsters []MonsterStats `json:"monsters"`
}

type JoinToStartedGameEvent struct {
	GameData map[string]interface{} `json:"gameData"`
}

type PlayerDeathEvent struct {
	ClientID uint64 `json:"clientId"`
}

type ChestOpenEvent struct {
	ObjectID int `json:"objectId"`
}

type KeyCollectedEvent struct {
	Number string `json:"number"`
}

type TileData struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	TileID int `json:"tileId"`
}

type UpdateTilesEvent struct {
	LayerName string     `json:"layerName"`
	Tiles     []TileData `json:"tiles"`
}
