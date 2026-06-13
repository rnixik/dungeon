package game

import (
	"dungeon/internal/lobby"
	"encoding/json"
	"log"
)

type BotClient struct {
	lobby.Client
	id                   uint64
	incomingEvents       chan interface{}
	outgoingCommands     chan *GameBotCommandWithName
	sendGameCommand      func(client lobby.ClientPlayer, commandType string, commandData json.RawMessage)
	stopped              bool
	additionalProperties map[string]interface{}
}

type GameBotCommandWithName struct {
	commandName string
	commandData json.RawMessage
}

type BotClientCommandEncodeWrapper struct {
	Data interface{} `json:"data"`
}

type BotClientCommandDecodeWrapper struct {
	Data json.RawMessage `json:"data"`
}

func NewBotClient(botId uint64, room *lobby.Room, sendGameCommand func(client lobby.ClientPlayer, commandName string, commandData json.RawMessage)) lobby.ClientPlayer {
	botClient := &BotClient{
		id:               botId,
		incomingEvents:   make(chan interface{}, 256),
		outgoingCommands: make(chan *GameBotCommandWithName),
		sendGameCommand:  sendGameCommand,
	}
	botClient.SetNickname("BotClient")

	bot := newBot(botClient, room)
	go botClient.sendingCommandsToGame()
	go bot.run()

	return botClient
}

func (bc *BotClient) SendEvent(event interface{}) {
	if bc.stopped {
		return
	}
	// Bots consume the original event object, not the wire bytes.
	if pre, ok := event.(*lobby.PreEncodedEvent); ok {
		event = pre.Event
	}
	// Never block the broadcaster (which holds no lock, but other callers may):
	// drop the event if the bot is not keeping up, mirroring the transport client.
	select {
	case bc.incomingEvents <- event:
	default:
	}
}

func (bc *BotClient) sendCommandToGame(commandType string, commandData interface{}) {
	if bc.stopped {
		return
	}
	// Game accepts type json.RawMessage for data, because it comes from web clients.
	// To satisfy interface bot client should get json.RawMessage for commandData.
	// To achieve this we encode to json and decode data back.
	commandDataEncoded, err := json.Marshal(&BotClientCommandEncodeWrapper{commandData})
	if err != nil {
		log.Println("cannot encode bot command with type = "+commandType, err)
		return
	}
	var commandDataDecoded BotClientCommandDecodeWrapper
	err = json.Unmarshal(commandDataEncoded, &commandDataDecoded)
	if err != nil {
		log.Println("cannot decode back bot command with type = "+commandType, err)
		return
	}
	bc.outgoingCommands <- &GameBotCommandWithName{commandType, commandDataDecoded.Data}
}

func (bc *BotClient) ID() uint64 {
	return bc.id
}

func (bc *BotClient) sendingCommandsToGame() {
	for {
		select {
		case botCommandWithName := <-bc.outgoingCommands:
			bc.sendGameCommand(bc, botCommandWithName.commandName, botCommandWithName.commandData)
		}
	}
}

func (bc *BotClient) CloseConnection() {
	bc.stopped = true
}

func (c *BotClient) SetAdditionalProperties(properties map[string]interface{}) {
	c.additionalProperties = properties
}

func (c *BotClient) GetAdditionalProperties() map[string]interface{} {
	return c.additionalProperties
}
