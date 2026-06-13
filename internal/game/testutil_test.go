package game

// fakeClient is a test double for lobby.ClientPlayer. It records the events sent
// to it so tests can assert on server -> client messages.
type fakeClient struct {
	id         uint64
	nickname   string
	props      map[string]interface{}
	sentEvents []interface{}
}

func newFakeClient(id uint64) *fakeClient {
	return &fakeClient{
		id:       id,
		nickname: "player",
		props:    map[string]interface{}{},
	}
}

func (c *fakeClient) SendEvent(event interface{}) { c.sentEvents = append(c.sentEvents, event) }
func (c *fakeClient) ID() uint64                  { return c.id }
func (c *fakeClient) SetNickname(n string)        { c.nickname = n }
func (c *fakeClient) Nickname() string            { return c.nickname }
func (c *fakeClient) CloseConnection()            {}
func (c *fakeClient) GetAdditionalProperties() map[string]interface{} {
	return c.props
}
func (c *fakeClient) SetAdditionalProperties(p map[string]interface{}) { c.props = p }

// newTestGame builds a Game with an in-memory broadcast recorder and no map.
// Tests that exercise the *Unsafe damage/XP helpers don't need a real map, room
// or main loop. The returned slice pointer collects all broadcast events.
func newTestGame() (*Game, *[]interface{}) {
	broadcast := &[]interface{}{}
	g := &Game{
		status:             StatusStarted,
		players:            make(map[uint64]*Player),
		monsters:           []*Monster{},
		objects:            make(map[uint64]*Object),
		traps:              make(map[string]*Trap),
		keysCollected:      map[string]bool{},
		broadcastEventFunc: func(event interface{}) { *broadcast = append(*broadcast, event) },
	}

	return g, broadcast
}

// addTestPlayer registers a player of the given class with full HP and returns it
// along with its fake client.
func addTestPlayer(g *Game, id uint64, class string) (*Player, *fakeClient) {
	client := newFakeClient(id)
	maxHP := classMaxHP(class)
	p := &Player{
		client:      client,
		class:       class,
		level:       1,
		nextLevelXP: 500,
		maxHp:       maxHP,
		hp:          maxHP,
		direction:   "right",
	}
	g.players[id] = p

	return p, client
}
