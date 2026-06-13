package lobby

import "testing"

func findEvent[T any](events []interface{}) (T, bool) {
	for _, e := range events {
		if typed, ok := e.(T); ok {
			return typed, true
		}
	}
	var zero T
	return zero, false
}

func TestJoinLobbyCommand(t *testing.T) {
	l, _, _ := newTestLobby(1, 2)
	c := newFakeClient(1, "")
	l.clients[c.ID()] = c

	l.joinLobbyCommand(c, "Alice")

	if c.Nickname() != "Alice" {
		t.Errorf("nickname = %q, want Alice", c.Nickname())
	}
	// Client receives a personal ClientJoinedEvent.
	joined, ok := findEvent[*ClientJoinedEvent](c.sentEvents)
	if !ok {
		t.Fatal("expected ClientJoinedEvent sent to client")
	}
	if joined.YourId != 1 || joined.YourNickname != "Alice" {
		t.Errorf("ClientJoinedEvent = %+v, want id 1 / Alice", joined)
	}
	if len(joined.Clients) != 1 {
		t.Errorf("expected 1 client in list, got %d", len(joined.Clients))
	}
	// Everyone is told about the join.
	if _, ok := findEvent[*ClientBroadCastJoinedEvent](drainBroadcast(l)); !ok {
		t.Error("expected ClientBroadCastJoinedEvent broadcast")
	}
}

func TestCreateNewRoomCommand(t *testing.T) {
	l, _, _ := newTestLobby(1, 2)
	c := newFakeClient(1, "owner")
	l.clients[c.ID()] = c

	room := l.CreateNewRoomCommand(c)
	if room == nil {
		t.Fatal("expected a room to be created")
	}
	if l.roomsCreatedByClients[c] != room {
		t.Error("room not registered under its owner")
	}
	if l.clientsJoinedRooms[c] != room {
		t.Error("owner not marked as joined to the room")
	}
	if _, ok := findEvent[*RoomJoinedEvent](c.sentEvents); !ok {
		t.Error("expected RoomJoinedEvent sent to owner")
	}
	if _, ok := findEvent[*ClientCreatedRoomEvent](drainBroadcast(l)); !ok {
		t.Error("expected ClientCreatedRoomEvent broadcast")
	}
}

func TestCreateNewRoomCommandOnlyOnePerClient(t *testing.T) {
	l, _, _ := newTestLobby(1, 2)
	c := newFakeClient(1, "owner")
	l.clients[c.ID()] = c

	l.CreateNewRoomCommand(c)
	c.sentEvents = nil
	second := l.CreateNewRoomCommand(c)

	if second != nil {
		t.Error("expected second room creation to be rejected")
	}
	errEvent, ok := findEvent[*ClientCommandError](c.sentEvents)
	if !ok || errEvent.Message != errorYouCanCreateOneRoomOnly {
		t.Errorf("expected %q error, got %v", errorYouCanCreateOneRoomOnly, c.sentEvents)
	}
}

func TestGetRoomById(t *testing.T) {
	l, _, _ := newTestLobby(1, 2)
	c := newFakeClient(1, "owner")
	room := l.CreateNewRoomCommand(c)

	got, err := l.getRoomById(room.ID())
	if err != nil || got != room {
		t.Errorf("getRoomById(%d) = %v, %v; want the room", room.ID(), got, err)
	}

	if _, err := l.getRoomById(99999); err == nil {
		t.Error("expected error for unknown room id")
	}
}

func TestJoinRoomCommand(t *testing.T) {
	l, _, _ := newTestLobby(1, 2)
	owner := newFakeClient(1, "owner")
	joiner := newFakeClient(2, "joiner")
	room := l.CreateNewRoomCommand(owner)
	drainBroadcast(l)

	l.JoinRoomCommand(joiner, room.ID())

	if l.clientsJoinedRooms[joiner] != room {
		t.Error("joiner not recorded as joined to the room")
	}
	if _, ok := room.getRoomMember(joiner); !ok {
		t.Error("joiner not added to room members")
	}
	if _, ok := findEvent[*RoomJoinedEvent](joiner.sentEvents); !ok {
		t.Error("expected RoomJoinedEvent sent to joiner")
	}
	if _, ok := findEvent[*RoomInListUpdatedEvent](drainBroadcast(l)); !ok {
		t.Error("expected RoomInListUpdatedEvent broadcast")
	}
}

func TestJoinRoomCommandUnknownRoom(t *testing.T) {
	l, _, _ := newTestLobby(1, 2)
	c := newFakeClient(1, "x")

	l.JoinRoomCommand(c, 12345)

	if l.clientsJoinedRooms[c] != nil {
		t.Error("client should not be joined to a nonexistent room")
	}
	errEvent, ok := findEvent[*ClientCommandError](c.sentEvents)
	if !ok || errEvent.Message != errorRoomDoesNotExist {
		t.Errorf("expected %q error, got %v", errorRoomDoesNotExist, c.sentEvents)
	}
}

func TestJoinRoomCommandSameRoomIsNoOp(t *testing.T) {
	l, _, _ := newTestLobby(1, 2)
	owner := newFakeClient(1, "owner")
	room := l.CreateNewRoomCommand(owner)
	drainBroadcast(l)
	before := len(room.members)

	l.JoinRoomCommand(owner, room.ID()) // already in this room

	if len(room.members) != before {
		t.Errorf("member count changed on no-op rejoin: %d -> %d", before, len(room.members))
	}
	if len(drainBroadcast(l)) != 0 {
		t.Error("no-op rejoin should not broadcast")
	}
}

func TestOnClientLeftCancelsMatchAndLeavesRoom(t *testing.T) {
	l, mm, _ := newTestLobby(1, 2)
	owner := newFakeClient(1, "owner")
	room := l.CreateNewRoomCommand(owner)
	drainBroadcast(l)

	l.onClientLeft(owner)

	if len(mm.cancelled) != 1 || mm.cancelled[0] != owner {
		t.Error("expected matchMaker.Cancel to be called for the leaving client")
	}
	// Owner was the only member, so the room becomes empty and is removed.
	if len(mm.roomsRemoved) != 1 || mm.roomsRemoved[0] != room {
		t.Error("expected matchMaker.OnRoomRemoved for the emptied room")
	}
	if _, exists := l.roomsCreatedByClients[owner]; exists {
		t.Error("emptied room should be removed from roomsCreatedByClients")
	}
	events := drainBroadcast(l)
	if _, ok := findEvent[*RoomInListRemovedEvent](events); !ok {
		t.Error("expected RoomInListRemovedEvent broadcast")
	}
	if _, ok := findEvent[*ClientLeftEvent](events); !ok {
		t.Error("expected ClientLeftEvent broadcast")
	}
}

func TestOnClientCommandRoutesLobbyJoin(t *testing.T) {
	l, _, _ := newTestLobby(1, 2)
	c := newFakeClient(1, "")
	l.clients[c.ID()] = c

	l.onClientCommand(&ClientCommand{
		Type:    ClientCommandTypeLobby,
		SubType: ClientCommandLobbySubTypeJoin,
		Data:    mustJSON("Bob"),
		client:  c,
	})

	if c.Nickname() != "Bob" {
		t.Errorf("nickname = %q, want Bob (join command not routed)", c.Nickname())
	}
}

func TestOnClientCommandMakeMatchRouted(t *testing.T) {
	l, mm, _ := newTestLobby(1, 2)
	c := newFakeClient(1, "")

	l.onClientCommand(&ClientCommand{
		Type:    ClientCommandTypeLobby,
		SubType: ClientCommandLobbySubTypeMakeMatch,
		Data:    mustJSON(map[string]interface{}{"roomName": "abc"}),
		client:  c,
	})

	if mm.matchCalls != 1 {
		t.Errorf("matchMaker.MakeMatch calls = %d, want 1", mm.matchCalls)
	}
}

func TestOnClientCommandRoomRequiresJoinedRoom(t *testing.T) {
	l, _, _ := newTestLobby(1, 2)
	c := newFakeClient(1, "")

	// Should not panic when the client is not in any room.
	l.onClientCommand(&ClientCommand{
		Type:    ClientCommandTypeRoom,
		SubType: ClientCommandRoomSubTypeWantToPlay,
		client:  c,
	})
}
