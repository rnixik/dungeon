package lobby

import (
	"encoding/json"
	"log"
	"sync"
	"sync/atomic"
)

// RoomMember represents connected to a room client.
type RoomMember struct {
	client      ClientPlayer
	wantsToPlay bool
	isPlayer    bool
	isBot       bool
}

// Room represents place where some of the members want to start a new game.
type Room struct {
	id          uint64
	owner       *RoomMember
	members     map[*RoomMember]bool
	game        GameEventsDispatcher
	lobby       *Lobby
	membersLock sync.RWMutex
}

func newRoom(roomId uint64, owner ClientPlayer, lobby *Lobby) *Room {
	members := make(map[*RoomMember]bool)
	ownerInRoom := newRoomMember(owner, false)
	ownerInRoom.isPlayer = true
	members[ownerInRoom] = true
	room := &Room{roomId, ownerInRoom, members, nil, lobby, sync.RWMutex{}}
	lobby.clientsJoinedRooms[owner] = room

	return room
}

func newRoomMember(client ClientPlayer, isBot bool) *RoomMember {
	return &RoomMember{client, true, false, isBot}
}

// Name returns name of the room by its owner.
func (r *Room) Name() string {
	return r.owner.client.Nickname()
}

// ID returns id of the room
func (r *Room) ID() uint64 {
	return r.id
}

// Game returns current game instance in the room
func (r *Room) Game() GameEventsDispatcher {
	return r.game
}

func (r *Room) getRoomMember(client ClientPlayer) (*RoomMember, bool) {
	for c := range r.members {
		if c.client.ID() == client.ID() {
			return c, true
		}
	}
	return nil, false
}

func (r *Room) removeClient(client ClientPlayer) (changedOwner bool, roomBecameEmpty bool) {
	member, ok := r.getRoomMember(client)
	if !ok {
		return
	}

	r.membersLock.Lock()
	delete(r.members, member)
	r.membersLock.Unlock()

	if r.game != nil {
		r.game.OnClientRemoved(client)
	}

	roomUpdatedEvent := &RoomUpdatedEvent{r.toRoomInfo(), RoomUpdatedCauseClientRemoved}
	r.broadcastEvent(roomUpdatedEvent, client)

	nonBotsMembersNumber := 0
	for ic := range r.members {
		if !ic.isBot {
			nonBotsMembersNumber += 1
		}
	}

	if nonBotsMembersNumber == 0 {
		roomBecameEmpty = true
		return
	}
	if r.owner == member {
		for ic := range r.members {
			r.owner = ic
			changedOwner = true
			return
		}
	}
	return
}

func (r *Room) addClient(client ClientPlayer) {
	member := newRoomMember(client, false)
	r.members[member] = true

	if len(r.getPlayers()) < 2 {
		member.isPlayer = true
	}

	roomUpdatedEvent := &RoomUpdatedEvent{r.toRoomInfo(), RoomUpdatedCauseClientAdded}
	r.broadcastEvent(roomUpdatedEvent, member.client)

	roomJoinedEvent := &RoomJoinedEvent{r.toRoomInfo()}
	client.SendEvent(roomJoinedEvent)
}

func (r *Room) addBot(botClient ClientPlayer) {
	member := newRoomMember(botClient, true)
	r.members[member] = true
	member.isPlayer = true

	roomUpdatedEvent := &RoomUpdatedEvent{r.toRoomInfo(), RoomUpdatedCauseBotAdded}
	r.broadcastEvent(roomUpdatedEvent, nil)

	roomJoinedEvent := &RoomJoinedEvent{r.toRoomInfo()}
	botClient.SendEvent(roomJoinedEvent)
}

func (r *Room) broadcastEvent(event interface{}, exceptClient ClientPlayer) {
	r.membersLock.RLock()
	defer r.membersLock.RUnlock()

	for m := range r.members {
		if exceptClient == nil || m.client.ID() != exceptClient.ID() {
			m.client.SendEvent(event)
		}
	}
}

func (r *Room) getPlayers() []*RoomMember {
	players := make([]*RoomMember, 0)
	for rm := range r.members {
		if rm.isPlayer {
			players = append(players, rm)
		}
	}
	return players
}

func (r *Room) getMembersWhoWantToPlay() []*RoomMember {
	membersWhoWantToPlay := make([]*RoomMember, 0)
	for rm := range r.members {
		if rm.wantsToPlay {
			membersWhoWantToPlay = append(membersWhoWantToPlay, rm)
		}
	}
	return membersWhoWantToPlay
}

func (r *Room) hasSlotForPlayer() bool {
	membersWhoWantToPlayNum := 0
	for rm := range r.members {
		if rm.wantsToPlay {
			membersWhoWantToPlayNum++
		}
	}
	return membersWhoWantToPlayNum+1 <= r.lobby.maxPlayersInRoom
}

func (r *Room) changeMemberWantStatus(client ClientPlayer, wantsToPlay bool) {
	member, ok := r.getRoomMember(client)
	if !ok {
		return
	}
	member.wantsToPlay = wantsToPlay
	memberInfo := member.memberToRoomMemberInfo()
	changeStatusEvent := &RoomMemberChangedStatusEvent{memberInfo}
	r.broadcastEvent(changeStatusEvent, nil)
}

func (r *Room) onWantToPlayCommand(client ClientPlayer) {
	if r.game != nil {
		errEvent := &ClientCommandError{errorCantChangeStatusGameHasBeenStarted}
		client.SendEvent(errEvent)
		return
	}
	r.changeMemberWantStatus(client, true)
}

func (r *Room) onWantToSpectateCommand(client ClientPlayer) {
	if r.game != nil {
		errEvent := &ClientCommandError{errorCantChangeStatusGameHasBeenStarted}
		client.SendEvent(errEvent)
		return
	}
	r.changeMemberWantStatus(client, false)
	r.setPlayerStatus(client.ID(), false)
}

func (r *Room) onSetPlayerStatusCommand(c ClientPlayer, memberId uint64, playerStatus bool) {
	if r.owner.client.ID() != c.ID() {
		errEvent := &ClientCommandError{errorYouShouldBeOwner}
		c.SendEvent(errEvent)
		return
	}
	if r.game != nil {
		errEvent := &ClientCommandError{errorCantChangeStatusGameHasBeenStarted}
		c.SendEvent(errEvent)
		return
	}

	if playerStatus && !r.hasSlotForPlayer() {
		errEvent := &ClientCommandError{errorNumberOfPlayersExceededLimit}
		c.SendEvent(errEvent)
		return
	}

	r.setPlayerStatus(memberId, playerStatus)
}

func (r *Room) setPlayerStatus(memberId uint64, playerStatus bool) {
	var foundMember *RoomMember
	for rm := range r.members {
		if rm.client.ID() == memberId {
			rm.isPlayer = playerStatus
			foundMember = rm
			break
		}
	}

	if foundMember == nil {
		return
	}

	memberInfo := foundMember.memberToRoomMemberInfo()
	roomMemberChangedPlayerStatusEvent := &RoomMemberChangedPlayerStatusEvent{memberInfo}
	r.broadcastEvent(roomMemberChangedPlayerStatusEvent, nil)
}

func (r *Room) OnStartGameCommand(c ClientPlayer) {
	if r.Game() != nil {
		log.Println("game has been already started in the room:", r.ID())
		r.Game().OnClientJoined(c)

		return
	}
	pls := r.getPlayers()
	if len(pls) < r.lobby.minPlayersInRoom {
		errEvent := &ClientCommandError{errorNeedMorePlayers}
		c.SendEvent(errEvent)
		return
	}
	if len(pls) > r.lobby.maxPlayersInRoom {
		errEvent := &ClientCommandError{errorNumberOfPlayersExceededLimit}
		c.SendEvent(errEvent)
		return
	}
	if r.game != nil {
		errEvent := &ClientCommandError{errorGameHasBeenAlreadyStarted}
		c.SendEvent(errEvent)
		return
	}

	playersClients := make([]ClientPlayer, 0)
	for rm := range r.members {
		if rm.isPlayer {
			playersClients = append(playersClients, rm.client)
		}
	}

	r.game = r.lobby.newGameFunc(playersClients, r, func(event interface{}) {
		r.broadcastEvent(event, nil)
	})
	go r.game.StartMainLoop()

	roomUpdatedEvent := &RoomUpdatedEvent{r.toRoomInfo(), RoomUpdatedCauseGameStarted}
	r.broadcastEvent(roomUpdatedEvent, nil)

	gameStartedEvent := &GameStartedEvent{Room: r.toRoomInfo(), GameData: r.game.GetCommonInitialGameData()}
	r.broadcastEvent(gameStartedEvent, nil)

	r.lobby.sendRoomUpdate(r)
}

func (r *Room) onDeleteGameCommand(c ClientPlayer) {
	if r.owner.client.ID() != c.ID() {
		errEvent := &ClientCommandError{errorYouShouldBeOwner}
		c.SendEvent(errEvent)
		return
	}
	if r.game == nil {
		errEvent := &ClientCommandError{errorGameAlreadyDeleted}
		c.SendEvent(errEvent)
		return
	}

	r.game = nil

	roomUpdatedEvent := &RoomUpdatedEvent{r.toRoomInfo(), RoomUpdatedCauseGameDeleted}
	r.broadcastEvent(roomUpdatedEvent, nil)

	r.lobby.sendRoomUpdate(r)
}

func (r *Room) onAddBotCommand(c ClientPlayer) {
	if r.owner.client.ID() != c.ID() {
		errEvent := &ClientCommandError{errorYouShouldBeOwner}
		c.SendEvent(errEvent)
		return
	}
	if r.game != nil {
		errEvent := &ClientCommandError{errorGameHasBeenAlreadyStarted}
		c.SendEvent(errEvent)
		return
	}
	if !r.hasSlotForPlayer() {
		errEvent := &ClientCommandError{errorNumberOfPlayersExceededLimit}
		c.SendEvent(errEvent)
		return
	}

	r.CreateBot()
}

func (r *Room) CreateBot() ClientPlayer {
	atomic.AddUint64(&lastClientId, 1)
	lastBotIdSafe := atomic.LoadUint64(&lastClientId)
	clientPlayer := r.lobby.newBotFunc(lastBotIdSafe, r, func(client ClientPlayer, eventName string, eventData json.RawMessage) {
		if r.game == nil {
			return
		}
		r.game.DispatchGameCommand(client, eventName, eventData)
	})
	client := clientPlayer.(ClientPlayer)
	r.addBot(client)

	return client
}

func (r *Room) onRemoveBotsCommand(c ClientPlayer) {
	if r.owner.client.ID() != c.ID() {
		errEvent := &ClientCommandError{errorYouShouldBeOwner}
		c.SendEvent(errEvent)
		return
	}
	if r.game != nil {
		errEvent := &ClientCommandError{errorGameHasBeenAlreadyStarted}
		c.SendEvent(errEvent)
		return
	}

	for rm := range r.members {
		if rm.isBot {
			r.removeClient(rm.client)
		}
	}
}

func (r *Room) onClientCommand(cc *ClientCommand) {
	switch cc.SubType {
	case ClientCommandRoomSubTypeWantToPlay:
		r.onWantToPlayCommand(cc.client)
	case ClientCommandRoomSubTypeWantToSpectate:
		r.onWantToSpectateCommand(cc.client)
	case ClientCommandRoomSubTypeSetPlayerStatus:
		var statusData RoomSetPlayerStatusCommandData
		if err := json.Unmarshal(cc.Data, &statusData); err != nil {
			return
		}
		r.onSetPlayerStatusCommand(cc.client, statusData.MemberId, statusData.Status)
	case ClientCommandRoomSubTypeStartGame:
		r.OnStartGameCommand(cc.client)
	case ClientCommandRoomSubTypeDeleteGame:
		r.onDeleteGameCommand(cc.client)
	case ClientCommandRoomSubTypeAddBot:
		r.onAddBotCommand(cc.client)
	case ClientCommandRoomSubTypeRemoveBots:
		r.onRemoveBotsCommand(cc.client)
	case ClientCommandRoomSetAdditionalProperties:
		log.Println("setting additional properties for client:", cc.Data)
		var propertiesData map[string]interface{}
		if err := json.Unmarshal(cc.Data, &propertiesData); err != nil {
			return
		}
		cc.client.SetAdditionalProperties(propertiesData)
	}
}

func (r *Room) onGameStarted() {
	roomUpdatedEvent := &RoomUpdatedEvent{r.toRoomInfo(), RoomUpdatedCauseGameStarted}
	r.broadcastEvent(roomUpdatedEvent, nil)
	r.lobby.sendRoomUpdate(r)
}

func (r *Room) OnGameEnded() {
	for rm := range r.members {
		if rm.isBot {
			rm.client.CloseConnection()
			r.removeClient(rm.client)
		}
	}

	r.game = nil

	roomUpdatedEvent := &RoomUpdatedEvent{r.toRoomInfo(), RoomUpdatedCauseGameEnded}
	r.broadcastEvent(roomUpdatedEvent, nil)
	r.lobby.sendRoomUpdate(r)
}

func (rm *RoomMember) memberToRoomMemberInfo() *RoomMemberInfo {
	return &RoomMemberInfo{
		Id:          rm.client.ID(),
		Nickname:    rm.client.Nickname(),
		WantsToPlay: rm.wantsToPlay,
		IsPlayer:    rm.isPlayer,
		IsBot:       rm.isBot,
	}
}

func (r *Room) toRoomInList() *RoomInList {
	gameStatus := ""
	if r.game != nil {
		gameStatus = r.game.Status()
	}
	roomInList := &RoomInList{
		Id:         r.ID(),
		OwnerId:    r.owner.client.ID(),
		Name:       r.Name(),
		GameStatus: gameStatus,
		MembersNum: len(r.members),
	}

	return roomInList
}

func (r *Room) toRoomInfo() *RoomInfo {
	gameStatus := ""
	if r.game != nil {
		gameStatus = r.game.Status()
	}

	membersInfo := make([]*RoomMemberInfo, 0)
	for member := range r.members {
		memberInfo := member.memberToRoomMemberInfo()
		membersInfo = append(membersInfo, memberInfo)
	}

	roomInfo := &RoomInfo{
		Id:         r.ID(),
		OwnerId:    r.owner.client.ID(),
		Name:       r.Name(),
		GameStatus: gameStatus,
		Members:    membersInfo,
		MaxPlayers: r.lobby.maxPlayersInRoom,
	}

	return roomInfo
}
