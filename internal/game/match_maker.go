package game

import (
	"dungeon/internal/lobby"
)

type MatchMaker struct {
	waitingClient *lobby.ClientPlayer
}

func NewMatchMaker() *MatchMaker {
	return &MatchMaker{}
}

func (mm *MatchMaker) MakeMatch(client lobby.ClientPlayer, foundFunc func(clientsIds []lobby.ClientPlayer), notFoundFunc func(), addBotFunc func() lobby.ClientPlayer) {
	// only with bots
	//botClient := addBotFunc()
	//foundFunc([]lobby.ClientPlayer{client, botClient})

	// game with players
	if mm.waitingClient != nil && (*mm.waitingClient).ID() != client.ID() {
		foundFunc([]lobby.ClientPlayer{*mm.waitingClient, client})
		mm.waitingClient = nil
		return
	}

	mm.waitingClient = &client
}

func (mm *MatchMaker) Cancel(client lobby.ClientPlayer) {
	if mm.waitingClient != nil && (*mm.waitingClient).ID() == client.ID() {
		mm.waitingClient = nil
	}
}
