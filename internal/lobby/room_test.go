package lobby

import "testing"

// makeRoom creates a room owned by a fresh client in a test lobby.
func makeRoom(l *Lobby, ownerID uint64) (*Room, *fakeClient) {
	owner := newFakeClient(ownerID, "owner")
	room := newRoom(1, owner, l)
	return room, owner
}

func TestNewRoom(t *testing.T) {
	l, _, _ := newTestLobby(1, 2)
	room, owner := makeRoom(l, 1)

	if room.Name() != "owner" {
		t.Errorf("Name() = %q, want owner", room.Name())
	}
	if len(room.members) != 1 {
		t.Errorf("member count = %d, want 1", len(room.members))
	}
	if l.clientsJoinedRooms[owner] != room {
		t.Error("owner not recorded as joined in the lobby")
	}
	players := room.getPlayers()
	if len(players) != 1 || players[0].client.ID() != owner.ID() {
		t.Error("owner should be the sole player")
	}
}

func TestAddClientPlayerSlots(t *testing.T) {
	l, _, _ := newTestLobby(1, 3)
	room, _ := makeRoom(l, 1)

	second := newFakeClient(2, "second")
	room.addClient(second)
	if m, _ := room.getRoomMember(second); !m.isPlayer {
		t.Error("second member should become a player (under 2 players)")
	}
	if _, ok := findEvent[*RoomJoinedEvent](second.sentEvents); !ok {
		t.Error("added client should receive RoomJoinedEvent")
	}

	third := newFakeClient(3, "third")
	room.addClient(third)
	if m, _ := room.getRoomMember(third); m.isPlayer {
		t.Error("third member should NOT be a player (already 2 players)")
	}
}

func TestRemoveClientOwnerHandover(t *testing.T) {
	l, _, _ := newTestLobby(1, 3)
	room, owner := makeRoom(l, 1)
	second := newFakeClient(2, "second")
	room.addClient(second)

	changedOwner, becameEmpty := room.removeClient(owner)
	if !changedOwner {
		t.Error("expected owner handover when owner leaves with others present")
	}
	if becameEmpty {
		t.Error("room should not be empty while a member remains")
	}
	if room.owner.client.ID() != second.ID() {
		t.Errorf("new owner ID = %d, want 2", room.owner.client.ID())
	}
}

func TestRemoveClientLastMemberEmptiesRoom(t *testing.T) {
	l, _, _ := newTestLobby(1, 3)
	room, owner := makeRoom(l, 1)

	_, becameEmpty := room.removeClient(owner)
	if !becameEmpty {
		t.Error("expected room to become empty when the last member leaves")
	}
}

func TestRemoveClientNotAMember(t *testing.T) {
	l, _, _ := newTestLobby(1, 3)
	room, _ := makeRoom(l, 1)
	stranger := newFakeClient(99, "stranger")

	changedOwner, becameEmpty := room.removeClient(stranger)
	if changedOwner || becameEmpty {
		t.Error("removing a non-member should be a no-op")
	}
}

func TestHasSlotForPlayer(t *testing.T) {
	l, _, _ := newTestLobby(1, 2)
	room, _ := makeRoom(l, 1) // owner wantsToPlay -> 1 wanting; +1 = 2 <= 2

	if !room.hasSlotForPlayer() {
		t.Error("expected a free slot with one player and max 2")
	}

	second := newFakeClient(2, "second")
	room.addClient(second) // now 2 members wanting; +1 = 3 > 2
	if room.hasSlotForPlayer() {
		t.Error("expected no free slot once the room is full")
	}
}

func TestChangeMemberWantStatus(t *testing.T) {
	l, _, _ := newTestLobby(1, 2)
	room, owner := makeRoom(l, 1)

	room.onWantToSpectateCommand(owner)
	m, _ := room.getRoomMember(owner)
	if m.wantsToPlay {
		t.Error("wantToSpectate should clear wantsToPlay")
	}
	if m.isPlayer {
		t.Error("wantToSpectate should clear player status")
	}

	room.onWantToPlayCommand(owner)
	m, _ = room.getRoomMember(owner)
	if !m.wantsToPlay {
		t.Error("wantToPlay should set wantsToPlay")
	}
}

func TestSetPlayerStatusRequiresOwner(t *testing.T) {
	l, _, _ := newTestLobby(1, 3)
	room, _ := makeRoom(l, 1)
	second := newFakeClient(2, "second")
	room.addClient(second)

	// Non-owner attempt is rejected.
	room.onSetPlayerStatusCommand(second, second.ID(), false)
	errEvent, ok := findEvent[*ClientCommandError](second.sentEvents)
	if !ok || errEvent.Message != errorYouShouldBeOwner {
		t.Errorf("expected %q error for non-owner, got %v", errorYouShouldBeOwner, second.sentEvents)
	}
}

func TestToRoomInfoAndInList(t *testing.T) {
	l, _, _ := newTestLobby(1, 2)
	room, owner := makeRoom(l, 1)

	info := room.toRoomInfo()
	if info.Id != room.ID() || info.OwnerId != owner.ID() {
		t.Errorf("RoomInfo ids = %d/%d, want %d/%d", info.Id, info.OwnerId, room.ID(), owner.ID())
	}
	if info.GameStatus != "" {
		t.Errorf("GameStatus = %q, want empty (no game)", info.GameStatus)
	}
	if info.MaxPlayers != 2 {
		t.Errorf("MaxPlayers = %d, want 2", info.MaxPlayers)
	}
	if len(info.Members) != 1 {
		t.Errorf("Members = %d, want 1", len(info.Members))
	}

	list := room.toRoomInList()
	if list.MembersNum != 1 || list.OwnerId != owner.ID() {
		t.Errorf("RoomInList = %+v, want 1 member / owner %d", list, owner.ID())
	}
}

func TestOnStartGameCommandNeedsMorePlayers(t *testing.T) {
	l, _, _ := newTestLobby(2, 4) // requires at least 2 players
	room, owner := makeRoom(l, 1) // only the owner is a player

	room.OnStartGameCommand(owner)

	if room.game != nil {
		t.Error("game should not start with too few players")
	}
	errEvent, ok := findEvent[*ClientCommandError](owner.sentEvents)
	if !ok || errEvent.Message != errorNeedMorePlayers {
		t.Errorf("expected %q error, got %v", errorNeedMorePlayers, owner.sentEvents)
	}
}

func TestOnStartGameCommandStartsGame(t *testing.T) {
	l, _, game := newTestLobby(1, 4)
	room, owner := makeRoom(l, 1)

	room.OnStartGameCommand(owner)

	if room.game == nil {
		t.Fatal("expected a game to be created")
	}
	// StartMainLoop runs in a goroutine; wait for it to signal.
	<-game.loopStarted

	events := drainBroadcast(l)
	if _, ok := findEvent[*RoomInListUpdatedEvent](events); !ok {
		t.Error("expected a lobby room update broadcast on game start")
	}
}

func TestOnStartGameCommandAlreadyStartedJoinsClient(t *testing.T) {
	l, _, game := newTestLobby(1, 4)
	room, owner := makeRoom(l, 1)
	room.OnStartGameCommand(owner)
	<-game.loopStarted

	latecomer := newFakeClient(2, "late")
	room.OnStartGameCommand(latecomer)

	if len(game.clientsJoined) != 1 || game.clientsJoined[0] != latecomer {
		t.Error("expected late client to be joined to the running game")
	}
}

func TestOnClientCommandSetAdditionalProperties(t *testing.T) {
	l, _, _ := newTestLobby(1, 2)
	room, owner := makeRoom(l, 1)

	room.onClientCommand(&ClientCommand{
		SubType: ClientCommandRoomSetAdditionalProperties,
		Data:    mustJSON(map[string]interface{}{"class": "mage"}),
		client:  owner,
	})

	if owner.props["class"] != "mage" {
		t.Errorf("additional properties not set: %v", owner.props)
	}
}
