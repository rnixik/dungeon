package lobby

import (
	"encoding/json"
	"fmt"
	"log"
	"sync/atomic"
)

var lastClientId uint64
var lastRoomId uint64

type GameEventsDispatcher interface {
	DispatchGameCommand(client ClientPlayer, eventName string, eventData interface{})
	OnClientRemoved(client ClientPlayer)
	OnClientJoined(client ClientPlayer)
	StartMainLoop()
	Status() string
	GetCommonInitialGameData() map[string]interface{}
}

type NewGameFunc func(playersClients []ClientPlayer, room *Room, broadcastEventFunc func(event interface{})) GameEventsDispatcher
type NewBotFunc func(botId uint64, room *Room, sendGameEvent func(client ClientPlayer, eventName string, eventData json.RawMessage)) ClientPlayer

type MatchMakerSettings map[string]interface{}

type MatchMaker interface {
	MakeMatch(
		lobby *Lobby,
		client *ClientPlayer,
		settings MatchMakerSettings,
	)
	Cancel(client ClientPlayer)
	OnRoomRemoved(room *Room)
}

// Lobby is the first place for connected clients. It passes commands to games.
type Lobby struct {
	// Registered clients.
	clients map[uint64]ClientPlayer

	// Outbound events to the clients.
	broadcast chan interface{}

	// Register requests from the clients.
	register chan ClientSender

	// Unregister requests from clients.
	unregister chan ClientSender

	// Commands from clients
	clientCommands chan *ClientCommand

	// Started games
	games []GameEventsDispatcher

	// Rooms created by clients
	roomsCreatedByClients map[ClientPlayer]*Room

	// Room where client is
	clientsJoinedRooms map[ClientPlayer]*Room

	newGameFunc      NewGameFunc
	newBotFunc       NewBotFunc
	matchMaker       MatchMaker
	minPlayersInRoom int
	maxPlayersInRoom int
}

func NewLobby(newGameFunc NewGameFunc, newBotFunc NewBotFunc, matchMaker MatchMaker, minPlayersInRoom int, maxPlayersInRoom int) *Lobby {
	return &Lobby{
		broadcast:             make(chan interface{}),
		register:              make(chan ClientSender),
		unregister:            make(chan ClientSender),
		clients:               make(map[uint64]ClientPlayer),
		clientCommands:        make(chan *ClientCommand),
		games:                 make([]GameEventsDispatcher, 0),
		roomsCreatedByClients: make(map[ClientPlayer]*Room),
		clientsJoinedRooms:    make(map[ClientPlayer]*Room),
		newGameFunc:           newGameFunc,
		newBotFunc:            newBotFunc,
		matchMaker:            matchMaker,
		minPlayersInRoom:      minPlayersInRoom,
		maxPlayersInRoom:      maxPlayersInRoom,
	}
}

func (l *Lobby) Run() {
	log.Println("Go lobby")

	go func() {
		for {
			select {
			case event, ok := <-l.broadcast:
				if !ok {
					continue
				}
				for _, client := range l.clients {
					client.SendEvent(event)
				}
			}
		}
	}()

	for {
		select {
		case tc := <-l.register:
			atomic.AddUint64(&lastClientId, 1)
			lastClientIdSafe := atomic.LoadUint64(&lastClientId)
			tc.SetID(lastClientIdSafe)

			client := &Client{
				lobby:           l,
				transportClient: tc,
			}
			l.clients[client.ID()] = client
		case tc := <-l.unregister:
			if client, ok := l.clients[tc.ID()]; ok {
				client.CloseConnection()
				delete(l.clients, client.ID())
				l.onClientLeft(client)
			}
		case clientCommand := <-l.clientCommands:
			l.onClientCommand(clientCommand)
		}
	}
}

func (l *Lobby) RegisterTransportClient(tc ClientSender) {
	log.Println("Register transport client")
	l.register <- tc
	log.Println("Registered transport client")
}

func (l *Lobby) UnregisterTransportClient(tc ClientSender) {
	log.Println("Unregister transport client")
	l.unregister <- tc
	log.Println("Unregistered transport client")
}

func (l *Lobby) HandleClientCommand(tc ClientSender, clientCommand *ClientCommand) {
	if client, ok := l.clients[tc.ID()]; ok {
		clientCommand.client = client
		l.clientCommands <- clientCommand
	}
}

func (l *Lobby) broadcastEvent(event interface{}) {
	l.broadcast <- event
}

func (l *Lobby) joinLobbyCommand(c ClientPlayer, nickname string) {
	c.SetNickname(nickname)

	broadcastEvent := &ClientBroadCastJoinedEvent{
		Id:       c.ID(),
		Nickname: c.Nickname(),
	}
	l.broadcastEvent(broadcastEvent)

	clientsInList := make([]*ClientInList, 0)
	for _, client := range l.clients {
		clientInList := &ClientInList{
			Id:       client.ID(),
			Nickname: client.Nickname(),
		}
		clientsInList = append(clientsInList, clientInList)
	}

	roomsInList := make([]*RoomInList, 0)
	for _, room := range l.roomsCreatedByClients {
		roomInList := room.toRoomInList()
		roomsInList = append(roomsInList, roomInList)
	}

	event := &ClientJoinedEvent{
		YourId:       c.ID(),
		YourNickname: c.Nickname(),
		Clients:      clientsInList,
		Rooms:        roomsInList,
	}
	c.SendEvent(event)
}

func (l *Lobby) onClientLeft(client ClientPlayer) {
	l.matchMaker.Cancel(client)
	room := l.clientsJoinedRooms[client]
	if room != nil {
		l.onLeftRoom(client, room)
	}
	leftEvent := &ClientLeftEvent{
		Id: client.ID(),
	}
	l.broadcastEvent(leftEvent)
}

func (l *Lobby) CreateNewRoomCommand(c ClientPlayer) *Room {
	_, roomExists := l.roomsCreatedByClients[c]
	if roomExists {
		errEvent := &ClientCommandError{errorYouCanCreateOneRoomOnly}
		c.SendEvent(errEvent)
		return nil
	}

	oldRoomJoined := l.clientsJoinedRooms[c]
	if oldRoomJoined != nil {
		l.onLeftRoom(c, oldRoomJoined)
	}

	atomic.AddUint64(&lastRoomId, 1)
	lastRoomIdSafe := atomic.LoadUint64(&lastRoomId)

	room := newRoom(lastRoomIdSafe, c, l)
	l.roomsCreatedByClients[c] = room

	event := &ClientCreatedRoomEvent{room.toRoomInList()}
	l.broadcastEvent(event)

	roomJoinedEvent := &RoomJoinedEvent{room.toRoomInfo()}
	c.SendEvent(roomJoinedEvent)

	return room
}

func (l *Lobby) getRoomById(roomId uint64) (room *Room, err error) {
	for _, r := range l.roomsCreatedByClients {
		if r.ID() == roomId {
			return r, nil
		}
	}
	return nil, fmt.Errorf("room not found by id = %d", roomId)
}

func (l *Lobby) onLeftRoom(c ClientPlayer, room *Room) {
	changedOwner, roomBecameEmpty := room.removeClient(c)
	delete(l.clientsJoinedRooms, c)
	if roomBecameEmpty {
		l.matchMaker.OnRoomRemoved(room)
		roomInListRemovedEvent := &RoomInListRemovedEvent{room.ID()}
		l.broadcastEvent(roomInListRemovedEvent)
		l.roomsCreatedByClients[c] = nil
		delete(l.roomsCreatedByClients, c)

		return
	}
	if changedOwner {
		l.roomsCreatedByClients[room.owner.client] = room
		delete(l.roomsCreatedByClients, c)
	}
	roomInListUpdatedEvent := &RoomInListUpdatedEvent{room.toRoomInList()}
	l.broadcastEvent(roomInListUpdatedEvent)
}

func (l *Lobby) JoinRoomCommand(c ClientPlayer, roomId uint64) {
	oldRoomJoined := l.clientsJoinedRooms[c]
	if oldRoomJoined != nil && oldRoomJoined.ID() == roomId {
		return
	}
	if oldRoomJoined != nil {
		l.onLeftRoom(c, oldRoomJoined)
	}
	room, err := l.getRoomById(roomId)
	if err == nil {
		l.clientsJoinedRooms[c] = room
		room.addClient(c)
		roomInListUpdatedEvent := &RoomInListUpdatedEvent{room.toRoomInList()}
		l.broadcastEvent(roomInListUpdatedEvent)
	} else {
		errEvent := &ClientCommandError{errorRoomDoesNotExist}
		c.SendEvent(errEvent)
	}
}

func (l *Lobby) makeMatch(c ClientPlayer, mmSettings MatchMakerSettings) {
	l.matchMaker.MakeMatch(
		l,
		&c,
		mmSettings,
	)
}

func (l *Lobby) onClientCommand(cc *ClientCommand) {
	if cc.Type == ClientCommandTypeLobby {
		if cc.SubType == ClientCommandLobbySubTypeJoin {
			var nickname string
			if err := json.Unmarshal(cc.Data, &nickname); err != nil {
				return
			}
			l.joinLobbyCommand(cc.client, nickname)
		} else if cc.SubType == ClientCommandLobbySubTypeCreateRoom {
			l.CreateNewRoomCommand(cc.client)
		} else if cc.SubType == ClientCommandLobbySubTypeJoinRoom {
			var roomId uint64
			if err := json.Unmarshal(cc.Data, &roomId); err != nil {
				return
			}
			l.JoinRoomCommand(cc.client, roomId)
		} else if cc.SubType == ClientCommandLobbySubTypeMakeMatch {
			var mmSettings MatchMakerSettings
			if err := json.Unmarshal(cc.Data, &mmSettings); err != nil {
				return
			}
			l.makeMatch(cc.client, mmSettings)
		}
	} else if cc.Type == ClientCommandTypeRoom {
		if l.clientsJoinedRooms[cc.client] == nil {
			return
		}
		l.clientsJoinedRooms[cc.client].onClientCommand(cc)
	} else if cc.Type == ClientCommandTypeGame {
		l.dispatchGameCommand(cc)
	}
}

func (l *Lobby) dispatchGameCommand(cc *ClientCommand) {
	if l.clientsJoinedRooms[cc.client] == nil {
		return
	}
	if l.clientsJoinedRooms[cc.client].game == nil {
		return
	}
	l.clientsJoinedRooms[cc.client].game.DispatchGameCommand(cc.client, cc.SubType, cc.Data)
}

func (l *Lobby) sendRoomUpdate(room *Room) {
	roomInListUpdatedEvent := &RoomInListUpdatedEvent{room.toRoomInList()}
	l.broadcastEvent(roomInListUpdatedEvent)
}
