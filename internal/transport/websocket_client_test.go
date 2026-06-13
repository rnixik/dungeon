package transport

import (
	"encoding/json"
	"sync"
	"testing"
)

func TestWebSocketClientIDRoundTrip(t *testing.T) {
	c := &WebSocketClient{}
	c.SetID(42)
	if c.ID() != 42 {
		t.Errorf("ID() = %d, want 42", c.ID())
	}
}

func TestSendEventQueuesSerializedMessage(t *testing.T) {
	c := &WebSocketClient{send: make(chan []byte, 1), mu: sync.Mutex{}}

	c.SendEvent(&sampleEvent{Foo: "bar"})

	select {
	case msg := <-c.send:
		var decoded struct {
			Name string `json:"name"`
		}
		if err := json.Unmarshal(msg, &decoded); err != nil {
			t.Fatalf("queued message is not valid JSON: %v", err)
		}
		if decoded.Name != "sampleEvent" {
			t.Errorf("queued event name = %q, want sampleEvent", decoded.Name)
		}
	default:
		t.Fatal("expected a message queued on the send channel")
	}
}

func TestSendEventDropsWhenClosed(t *testing.T) {
	c := &WebSocketClient{send: make(chan []byte, 1), sendIsClosed: true, mu: sync.Mutex{}}

	c.SendEvent(&sampleEvent{Foo: "bar"})

	if len(c.send) != 0 {
		t.Error("SendEvent must not queue messages after the client is closed")
	}
}

func TestSendEventNonBlockingWhenFull(t *testing.T) {
	c := &WebSocketClient{send: make(chan []byte, 1), mu: sync.Mutex{}}
	c.send <- []byte("occupied") // fill the buffer

	done := make(chan struct{})
	go func() {
		c.SendEvent(&sampleEvent{Foo: "dropped"}) // must hit the default case, not block
		close(done)
	}()

	<-done // would deadlock if SendEvent blocked on a full channel
	if len(c.send) != 1 {
		t.Errorf("send buffer size = %d, want 1 (extra event dropped)", len(c.send))
	}
}
