package lobby

// RoomMemberInfo contains info about a client in the room
type RoomMemberInfo struct {
	Id          uint64 `json:"id"`
	Nickname    string `json:"nickname"`
	WantsToPlay bool   `json:"wantsToPlay"`
	IsPlayer    bool   `json:"isPlayer"`
	IsBot       bool   `json:"isBot"`
}

// RoomInfo contains info about room where client is.
type RoomInfo struct {
	Id         uint64            `json:"id"`
	OwnerId    uint64            `json:"ownerId"`
	Name       string            `json:"name"`
	GameStatus string            `json:"gameStatus"`
	Members    []*RoomMemberInfo `json:"members"`
	MaxPlayers int               `json:"maxPlayers"`
}

// RoomJoinedEvent contains info about room where client is
type RoomJoinedEvent struct {
	Room *RoomInfo `json:"room"`
}

// RoomUpdatedEvent contains info about updated room where client is
type RoomUpdatedEvent struct {
	Room  *RoomInfo `json:"room"`
	Cause string    `json:"cause"`
}

// GameStartedEvent broadcasted to all room members when game is started
type GameStartedEvent struct {
	Room *RoomInfo `json:"room"`
}

// RoomMemberChangedStatusEvent contains info about room member when he changes his status
type RoomMemberChangedStatusEvent struct {
	Room *RoomMemberInfo `json:"member"`
}

// RoomMemberChangedPlayerStatusEvent contains info about room member when his player status was changed by room owner
type RoomMemberChangedPlayerStatusEvent struct {
	Room *RoomMemberInfo `json:"member"`
}

// RoomSetPlayerStatusCommandData represents data from room owner to set or unset player status of a member
type RoomSetPlayerStatusCommandData struct {
	MemberId uint64 `json:"memberId"`
	Status   bool   `json:"status"`
}

const RoomUpdatedCauseBotAdded = "botAdded"
const RoomUpdatedCauseClientAdded = "clientAdded"
const RoomUpdatedCauseClientRemoved = "clientRemoved"
const RoomUpdatedCauseGameStarted = "gameStarted"
const RoomUpdatedCauseGameDeleted = "gameDeleted"
const RoomUpdatedCauseGameEnded = "gameEnded"
