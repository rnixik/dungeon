package lobby

const (
	errorNeedMorePlayers                    = "need_more_players"
	errorNumberOfPlayersExceededLimit       = "number_of_players_exceeded_limit"
	errorGameHasBeenAlreadyStarted          = "game_has_been_already_started"
	errorYouCanCreateOneRoomOnly            = "you_can_create_one_room_only"
	errorRoomDoesNotExist                   = "room_does_not_exist"
	errorCantChangeStatusGameHasBeenStarted = "cant_change_status_game_has_been_started"
	errorYouShouldBeOwner                   = "you_should_be_owner"
	errorGameAlreadyDeleted                 = "game_already_deleted"
)

// ClientCommandError contains info about error on client's command.
type ClientCommandError struct {
	Message string `json:"message"`
}
