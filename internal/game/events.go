package game

type DamageEvent struct {
	TargetPlayerId  uint64 `json:"targetPlayerId"`
	TargetMonsterID int    `json:"targetMonsterId"`
	Damage          int    `json:"damage"`
	X               int    `json:"x"` // Position for damage text
	Y               int    `json:"y"` // Position for damage text
}

type XPEvent struct {
	TargetPlayerId uint64 `json:"clientId"`
	XP             int    `json:"xp"`
	NextLevelXP    int    `json:"nextLevelXp"`
	Level          int    `json:"level"`
	GotNewLevel    bool   `json:"gotNewLevel"`
}

type FireballEvent struct {
	ClientID  uint64 `json:"clientId"`
	X         int    `json:"x"`
	Y         int    `json:"y"`
	Direction string `json:"direction"`
	Distance  int    `json:"distance"`
}

type ShootArrowEvent struct {
	ClientID uint64 `json:"clientId"`
	X1       int    `json:"x1"`
	Y1       int    `json:"y1"`
	X2       int    `json:"x2"`
	Y2       int    `json:"y2"`
	Velocity int    `json:"velocity"`
}

type SwordAttackPrepareEvent struct {
	ClientID  uint64 `json:"clientId"`
	X         int    `json:"x"`
	Y         int    `json:"y"`
	Direction string `json:"direction"`
}

type SwordAttackEvent struct {
	ClientID    uint64 `json:"clientId"`
	X           int    `json:"x"`
	Y           int    `json:"y"`
	AttackLineX int    `json:"attackLineX"`
	AttackLineY int    `json:"attackLineY"`
	Radius      int    `json:"radius"`
	Direction   string `json:"direction"`
}

type ArrowEvent struct {
	ClientID  uint64 `json:"clientId"`
	MonsterID int    `json:"monsterId"`
	X1        int    `json:"x1"`
	Y1        int    `json:"y1"`
	X2        int    `json:"x2"`
	Y2        int    `json:"y2"`
}

type SpawnSpikeEvent struct {
	X          int    `json:"x"`
	Y          int    `json:"y"`
	StartFrame string `json:"startFrame"`
}

type TrapStateChangedEvent struct {
	TrapID string    `json:"trapId"`
	State  TrapState `json:"state"`
	X      int       `json:"x"`
	Y      int       `json:"y"`
	Frame  int       `json:"frame"`
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
	TargetX   int `json:"targetX"`
	TargetY   int `json:"targetY"`
}

type GolemSlamEvent struct {
	MonsterID int `json:"monsterId"`
	X         int `json:"x"`
	Y         int `json:"y"`
	Radius    int `json:"radius"`
}

type SpiderWebEvent struct {
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
	IsDodging bool   `json:"isDodging"`
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
	Class             string `json:"class"`
	Nickname          string `json:"nickname"`
	AvatarUrl         string `json:"avatarUrl,omitempty"`
	Color             string `json:"color"`
	MaxHP             int    `json:"maxHp"`
	HP                int    `json:"hp"`
	Level             int    `json:"level"`
	XP                int    `json:"xp"`
	NextLevelXP       int    `json:"nextLevelXp"`
	SpeedBoostPercent int    `json:"speedBoostPercent"`
	HasShield         bool   `json:"hasShield"`
	IsInvisible       bool   `json:"isInvisible"`
}

type MonsterStats struct {
	MonsterPosition
	Kind  string `json:"kind"`
	HP    int    `json:"hp"`
	MaxHP int    `json:"maxHp"`
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
	Nickname string `json:"nickname"`
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

type PlayerRespawnEvent struct {
	ClientID uint64 `json:"clientId"`
	X        int    `json:"x"`
	Y        int    `json:"y"`
}

type InventoryItem struct {
	Kind       string `json:"kind"`
	Count      int    `json:"count"`
	CooldownMs int    `json:"cooldownMs,omitempty"`
}

type HealEvent struct {
	ClientID uint64 `json:"clientId"`
	Amount   int    `json:"amount"`
	HP       int    `json:"hp"`
	MaxHP    int    `json:"maxHp"`
}

type InventoryUpdateEvent struct {
	ClientID  uint64          `json:"clientId"`
	Inventory []InventoryItem `json:"inventory"`
}

type FootprintPoint struct {
	ClientID uint64 `json:"clientId"`
	X        int    `json:"x"`
	Y        int    `json:"y"`
	Color    string `json:"color"`
}

type FootprintsEvent struct {
	Points []FootprintPoint `json:"points"`
}

type FootprintsExpiredEvent struct{}

type JellySplitEvent struct {
	MonsterID int `json:"monsterID"`
	X         int `json:"x"`
	Y         int `json:"y"`
}

type JellyHitSlowEvent struct {
	Duration    int `json:"duration"`
	SlowPercent int `json:"slowPercent"`
}

type DemonMageShieldEvent struct {
	CasterID int `json:"casterId"`
	TargetID int `json:"targetId"`
	Duration int `json:"duration"` // milliseconds
}

type DemonMageSpeedBoostEvent struct {
	CasterID int `json:"casterId"`
	TargetID int `json:"targetId"`
	Duration int `json:"duration"` // milliseconds
}

type ProtectionActiveEvent struct {
	Duration int `json:"duration"` // milliseconds
}

type ProtectionExpiredEvent struct{}

type CloakActiveEvent struct {
	Duration   int `json:"duration"`   // milliseconds
	CooldownMs int `json:"cooldownMs"` // total cooldown in milliseconds
}

type CloakExpiredEvent struct{}

// SoulPowerEvent reports the Soul Power tally to a single client. Visible is
// true for cultists; for good players it is only true when debug is enabled.
type SoulPowerEvent struct {
	Value   int  `json:"value"`
	Visible bool `json:"visible"`
}

// BecameCultistEvent is sent only to the player who has just been cursed into a
// cultist so the client can reveal the curse text and switch to cultist vision.
type BecameCultistEvent struct{}

// CultistsRosterEvent is sent only to cultists so they can recognise each other.
// Good players never receive it.
type CultistsRosterEvent struct {
	ClientIDs []uint64 `json:"clientIds"`
}
