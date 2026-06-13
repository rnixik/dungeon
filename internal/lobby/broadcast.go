package lobby

// eventEncoder serializes an event to its wire bytes. It is set once by the
// transport layer at startup (see transport.init) so the lobby can pre-serialize
// a broadcast a single time instead of once per recipient — without importing
// transport, which would create an import cycle.
var eventEncoder func(event interface{}) ([]byte, error)

// SetEventEncoder registers the wire serializer used to pre-encode broadcasts.
func SetEventEncoder(enc func(event interface{}) ([]byte, error)) {
	eventEncoder = enc
}

// PreEncodedEvent carries an event together with its already-serialized wire
// bytes. broadcastEvent wraps an event in this once and passes it to every
// recipient: transport clients write Data directly (skipping a redundant
// per-client marshal), while non-transport clients such as bots unwrap Event to
// get the original object.
type PreEncodedEvent struct {
	Event interface{}
	Data  []byte
}
