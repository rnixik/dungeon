package lobby

// ClientSender represents interface which sends events to connected players.
type ClientSender interface {
	SendEvent(event interface{})
	ID() uint64
	SetID(id uint64)
	Close()
}

type ClientPlayer interface {
	SendEvent(event interface{})
	ID() uint64
	SetNickname(string)
	Nickname() string
	CloseConnection()
}

type Client struct {
	lobby *Lobby

	transportClient ClientSender

	nickname string
	room     *Room
}

func (c *Client) SendEvent(event interface{}) {
	c.transportClient.SendEvent(event)
}

func (c *Client) ID() uint64 {
	return c.transportClient.ID()
}

func (c *Client) SetNickname(nickname string) {
	// limit up to 24 chars
	c.nickname = nickname[:min(len(nickname), 24)]
}

func (c *Client) Nickname() string {
	return c.nickname
}

func (c *Client) CloseConnection() {
	c.transportClient.Close()
}
