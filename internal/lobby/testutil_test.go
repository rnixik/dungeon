package lobby

import "encoding/json"

// fakeClient is a test double for ClientPlayer that records sent events.
type fakeClient struct {
	id         uint64
	nickname   string
	props      map[string]interface{}
	sentEvents []interface{}
	closed     bool
}

func newFakeClient(id uint64, nickname string) *fakeClient {
	return &fakeClient{id: id, nickname: nickname, props: map[string]interface{}{}}
}

func (c *fakeClient) SendEvent(event interface{}) { c.sentEvents = append(c.sentEvents, event) }
func (c *fakeClient) ID() uint64                  { return c.id }
func (c *fakeClient) SetNickname(n string)        { c.nickname = n }
func (c *fakeClient) Nickname() string            { return c.nickname }
func (c *fakeClient) CloseConnection()            { c.closed = true }
func (c *fakeClient) GetAdditionalProperties() map[string]interface{} {
	return c.props
}
func (c *fakeClient) SetAdditionalProperties(p map[string]interface{}) { c.props = p }

// fakeSender is a test double for ClientSender (the transport-layer interface).
type fakeSender struct {
	id     uint64
	closed bool
	sent   []interface{}
}

func (s *fakeSender) SendEvent(event interface{}) { s.sent = append(s.sent, event) }
func (s *fakeSender) ID() uint64                  { return s.id }
func (s *fakeSender) SetID(id uint64)             { s.id = id }
func (s *fakeSender) Close()                      { s.closed = true }

// fakeMatchMaker records calls so lobby tests can assert routing without the
// real game-package matchmaker.
type fakeMatchMaker struct {
	matchCalls   int
	cancelled    []ClientPlayer
	roomsRemoved []*Room
}

func (m *fakeMatchMaker) MakeMatch(_ *Lobby, _ *ClientPlayer, _ MatchMakerSettings) {
	m.matchCalls++
}
func (m *fakeMatchMaker) Cancel(c ClientPlayer)    { m.cancelled = append(m.cancelled, c) }
func (m *fakeMatchMaker) OnRoomRemoved(room *Room) { m.roomsRemoved = append(m.roomsRemoved, room) }

// fakeGame is a test double for GameEventsDispatcher.
type fakeGame struct {
	status        string
	loopStarted   chan struct{}
	clientsJoined []ClientPlayer
	clientsRemvd  []ClientPlayer
}

func newFakeGame() *fakeGame {
	return &fakeGame{status: "started", loopStarted: make(chan struct{}, 1)}
}

func (g *fakeGame) DispatchGameCommand(_ ClientPlayer, _ string, _ interface{}) {}
func (g *fakeGame) OnClientRemoved(c ClientPlayer)                              { g.clientsRemvd = append(g.clientsRemvd, c) }
func (g *fakeGame) OnClientJoined(c ClientPlayer)                               { g.clientsJoined = append(g.clientsJoined, c) }
func (g *fakeGame) StartMainLoop()                                              { g.loopStarted <- struct{}{} }
func (g *fakeGame) Status() string                                              { return g.status }
func (g *fakeGame) GetCommonInitialGameData() map[string]interface{}            { return map[string]interface{}{} }

// newTestLobby builds a Lobby with buffered channels (so broadcastEvent never
// blocks without the Run() goroutine) and initialized maps. The returned
// matchMaker and createdGame let tests assert side effects.
func newTestLobby(minPlayers, maxPlayers int) (*Lobby, *fakeMatchMaker, *fakeGame) {
	mm := &fakeMatchMaker{}
	game := newFakeGame()
	l := &Lobby{
		broadcast:             make(chan interface{}, 256),
		register:              make(chan ClientSender, 1),
		unregister:            make(chan ClientSender, 1),
		clients:               make(map[uint64]ClientPlayer),
		clientCommands:        make(chan *ClientCommand, 1),
		roomsCreatedByClients: make(map[ClientPlayer]*Room),
		clientsJoinedRooms:    make(map[ClientPlayer]*Room),
		matchMaker:            mm,
		minPlayersInRoom:      minPlayers,
		maxPlayersInRoom:      maxPlayers,
		newGameFunc: func(_ []ClientPlayer, _ *Room, _ func(interface{})) GameEventsDispatcher {
			return game
		},
	}
	return l, mm, game
}

// drainBroadcast non-blockingly reads all queued broadcast events.
func drainBroadcast(l *Lobby) []interface{} {
	events := make([]interface{}, 0)
	for {
		select {
		case e := <-l.broadcast:
			events = append(events, e)
		default:
			return events
		}
	}
}

func mustJSON(v interface{}) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}
