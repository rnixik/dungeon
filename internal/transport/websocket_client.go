package transport

import (
	"dungeon/internal/lobby"
	"encoding/json"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"sync"
	"time"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// WebSocketClient represents a connected user using websockets.
type WebSocketClient struct {
	lobby *lobby.Lobby

	conn *websocket.Conn

	// Channel of outbound messages.
	send         chan []byte
	sendIsClosed bool
	mu           sync.Mutex

	id uint64
}

func (c *WebSocketClient) readLoop() {
	defer func() {
		c.lobby.UnregisterTransportClient(c)
		_ = c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))

	c.conn.SetPongHandler(func(string) error {
		_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway) {
				log.Printf("error: %v", err)
			}
			break
		}
		// log.Printf("Incoming message: %s", message)

		var clientCommand lobby.ClientCommand
		if err := json.Unmarshal(message, &clientCommand); err != nil {
			log.Printf("json unmarshal error: %s", err)
		} else {
			c.lobby.HandleClientCommand(c, &clientCommand)
		}
	}
}

func (c *WebSocketClient) writeLoop() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})

				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			_, _ = w.Write(message)

			if err2 := w.Close(); err2 != nil {
				c.Close()

				return
			}
		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (c *WebSocketClient) SendEvent(event interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	isClosed := c.sendIsClosed

	if isClosed {
		return
	}
	jsonDataMessage, _ := eventToJSON(event)
	if c.send == nil {
		return
	}
	c.send <- jsonDataMessage
}

func (c *WebSocketClient) SendMessage(message []byte) {

}

func (c *WebSocketClient) ID() uint64 {
	return c.id
}

func (c *WebSocketClient) SetID(id uint64) {
	c.id = id
}

func (c *WebSocketClient) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.sendIsClosed {
		return
	}

	c.sendIsClosed = true
	close(c.send)

	err := c.conn.Close()
	if err != nil {
		log.Println("Error closing websocket connection:", err)
	}
}

func ServeWebSocketRequest(lobby *lobby.Lobby, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	client := &WebSocketClient{
		lobby: lobby,
		conn:  conn,
		send:  make(chan []byte),
		mu:    sync.Mutex{},
	}
	client.lobby.RegisterTransportClient(client)

	go client.writeLoop()
	go client.readLoop()
}
