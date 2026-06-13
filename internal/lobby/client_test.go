package lobby

import (
	"strings"
	"testing"
)

func TestClientSetNicknameTruncates(t *testing.T) {
	c := &Client{transportClient: &fakeSender{id: 1}}

	c.SetNickname(strings.Repeat("a", 40))
	if len(c.Nickname()) != 24 {
		t.Errorf("nickname length = %d, want 24 (truncated)", len(c.Nickname()))
	}

	c.SetNickname("short")
	if c.Nickname() != "short" {
		t.Errorf("nickname = %q, want short", c.Nickname())
	}
}

func TestClientDelegatesToTransport(t *testing.T) {
	sender := &fakeSender{id: 7}
	c := &Client{transportClient: sender}

	if c.ID() != 7 {
		t.Errorf("ID() = %d, want 7", c.ID())
	}

	c.SendEvent("hello")
	if len(sender.sent) != 1 || sender.sent[0] != "hello" {
		t.Errorf("SendEvent did not delegate to transport: %v", sender.sent)
	}

	c.CloseConnection()
	if !sender.closed {
		t.Error("CloseConnection should close the transport client")
	}
}

func TestClientAdditionalProperties(t *testing.T) {
	c := &Client{transportClient: &fakeSender{id: 1}}
	props := map[string]interface{}{"class": "rogue"}

	c.SetAdditionalProperties(props)
	if got := c.GetAdditionalProperties(); got["class"] != "rogue" {
		t.Errorf("GetAdditionalProperties = %v, want class=rogue", got)
	}
}
