package game

import (
	"dungeon/internal/lobby"
)

type Bot struct {
	botClient          *BotClient
	room               *lobby.Room
	delayedCastCommand *CastCommand
}

func newBot(botClient *BotClient, room *lobby.Room) *Bot {
	return &Bot{
		botClient: botClient,
		room:      room,
	}
}

func (b *Bot) run() {

}
