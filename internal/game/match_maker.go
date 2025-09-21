package game

import (
	"dungeon/internal/lobby"
	"log"
)

type MatchMaker struct {
	roomByName map[string]*lobby.Room
}

func NewMatchMaker() *MatchMaker {
	return &MatchMaker{
		roomByName: make(map[string]*lobby.Room),
	}
}

func (mm *MatchMaker) MakeMatch(
	lobby *lobby.Lobby,
	client *lobby.ClientPlayer,
	settings lobby.MatchMakerSettings,
) {
	roomName, ok := settings["roomName"].(string)
	if !ok {
		roomName = "default"
	}
	room, ok := mm.roomByName[roomName]
	if ok {
		lobby.JoinRoomCommand(*client, room.ID())
	} else {
		room = lobby.CreateNewRoomCommand(*client)
		mm.roomByName[roomName] = room
	}

	if room == nil {
		log.Println("cannot create or join room")

		return
	}

	if room.Game() == nil {
		room.OnStartGameCommand(*client)
	}
}

func (mm *MatchMaker) Cancel(client lobby.ClientPlayer) {
}

func (mm *MatchMaker) OnRoomRemoved(room *lobby.Room) {
	for name, r := range mm.roomByName {
		if r.ID() == room.ID() {
			delete(mm.roomByName, name)
			break
		}
	}
}
