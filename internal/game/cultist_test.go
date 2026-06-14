package game

import "testing"

func TestGoodDeathBeforeBossFeedsSoulPower(t *testing.T) {
	g, _ := newTestGame()
	p, _ := addTestPlayer(g, 1, ClassKnight)

	g.killPlayer(1)

	if g.soulPower != 1 {
		t.Errorf("soulPower = %d, want 1", g.soulPower)
	}
	if p.goodDeathsBeforeBoss != 1 {
		t.Errorf("goodDeathsBeforeBoss = %d, want 1", p.goodDeathsBeforeBoss)
	}
}

func TestGoodDeathAfterBossDoesNotFeedSoulPower(t *testing.T) {
	g, _ := newTestGame()
	g.demonWasSpawned = true
	p, _ := addTestPlayer(g, 1, ClassKnight)

	g.killPlayer(1)

	if g.soulPower != 0 {
		t.Errorf("soulPower = %d, want 0 (no feed after boss)", g.soulPower)
	}
	if p.goodDeathsBeforeBoss != 0 {
		t.Errorf("goodDeathsBeforeBoss = %d, want 0", p.goodDeathsBeforeBoss)
	}
}

func TestCultistDeathDrainsSoulPower(t *testing.T) {
	g, _ := newTestGame()
	p, _ := addTestPlayer(g, 1, ClassRogue)
	p.isCultist = true

	g.killPlayer(1)

	if g.soulPower != -1 {
		t.Errorf("soulPower = %d, want -1", g.soulPower)
	}
}

func TestAlreadyDeadPlayerDoesNotDoubleCount(t *testing.T) {
	g, _ := newTestGame()
	p, _ := addTestPlayer(g, 1, ClassKnight)

	g.killPlayer(1)
	g.killPlayer(1) // already dead, must not count again

	if g.soulPower != 1 {
		t.Errorf("soulPower = %d, want 1 (no double count)", g.soulPower)
	}
	if p.goodDeathsBeforeBoss != 1 {
		t.Errorf("goodDeathsBeforeBoss = %d, want 1", p.goodDeathsBeforeBoss)
	}
}

func TestBecomingCultistUncountsItsGoodDeaths(t *testing.T) {
	g, _ := newTestGame()
	p, _ := addTestPlayer(g, 1, ClassMage)

	// Two good deaths before the boss feed +2.
	g.killPlayer(1)
	p.hp = p.maxHp
	g.killPlayer(1)
	if g.soulPower != 2 {
		t.Fatalf("soulPower = %d, want 2 before conversion", g.soulPower)
	}

	g.makePlayerCultistUnsafe(p)

	if !p.isCultist {
		t.Error("player should be a cultist")
	}
	if g.soulPower != 0 {
		t.Errorf("soulPower = %d, want 0 (this player's deaths uncounted)", g.soulPower)
	}
	if p.goodDeathsBeforeBoss != 0 {
		t.Errorf("goodDeathsBeforeBoss = %d, want 0 after conversion", p.goodDeathsBeforeBoss)
	}
}

func TestSoulPowerVisibilityPerPlayer(t *testing.T) {
	g, _ := newTestGame()
	cultist, cultistClient := addTestPlayer(g, 1, ClassMage)
	_, goodClient := addTestPlayer(g, 2, ClassKnight)
	cultist.isCultist = true

	g.broadcastSoulPowerUnsafe()

	lastVisible := func(c *fakeClient) (bool, bool) {
		for i := len(c.sentEvents) - 1; i >= 0; i-- {
			if e, ok := c.sentEvents[i].(SoulPowerEvent); ok {
				return e.Visible, true
			}
		}
		return false, false
	}

	if v, ok := lastVisible(cultistClient); !ok || !v {
		t.Errorf("cultist should see Soul Power (visible=%v, present=%v)", v, ok)
	}
	if v, ok := lastVisible(goodClient); !ok || v {
		t.Errorf("good player should not see Soul Power without debug (visible=%v)", v)
	}

	g.debug = true
	g.broadcastSoulPowerUnsafe()
	if v, ok := lastVisible(goodClient); !ok || !v {
		t.Errorf("good player should see Soul Power with debug on (visible=%v)", v)
	}
}

func TestCultistRosterOnlyToCultists(t *testing.T) {
	g, _ := newTestGame()
	cultist, cultistClient := addTestPlayer(g, 1, ClassMage)
	_, goodClient := addTestPlayer(g, 2, ClassKnight)
	cultist.isCultist = true

	g.broadcastCultistsRosterUnsafe()

	for _, e := range goodClient.sentEvents {
		if _, ok := e.(CultistsRosterEvent); ok {
			t.Error("good player must not receive a CultistsRosterEvent")
		}
	}
	var sawRoster bool
	for _, e := range cultistClient.sentEvents {
		if _, ok := e.(CultistsRosterEvent); ok {
			sawRoster = true
		}
	}
	if !sawRoster {
		t.Error("cultist should receive a CultistsRosterEvent")
	}
}

func TestGoodPlayerBecomesSpectatorAfterBoss(t *testing.T) {
	g, _ := newTestGame()
	g.demonWasSpawned = true
	p, client := addTestPlayer(g, 1, ClassKnight)

	g.killPlayer(1)
	if !p.isSpectator {
		t.Fatal("good player who died after boss should become a spectator")
	}

	g.respawnPlayer(1)
	if p.hp != 0 {
		t.Errorf("spectator must stay dead, hp = %d", p.hp)
	}
	if !hasRespawnDenied(client, respawnDeniedEliminated) {
		t.Error("expected RespawnDeniedEvent(eliminated)")
	}
}

func TestCultistRespawnAfterBossSpendsSoulPower(t *testing.T) {
	g, _ := newTestGame()
	g.demonWasSpawned = true
	g.soulPower = 2
	p, client := addTestPlayer(g, 1, ClassRogue)
	p.isCultist = true

	// First death after boss must not drain Soul Power; respawn spends 1.
	g.killPlayer(1)
	if g.soulPower != 2 {
		t.Fatalf("soulPower = %d, want 2 (cultist death after boss does not drain)", g.soulPower)
	}
	g.respawnPlayer(1)
	if p.hp != p.maxHp {
		t.Errorf("cultist should respawn, hp = %d", p.hp)
	}
	if g.soulPower != 1 {
		t.Errorf("soulPower = %d, want 1 after respawn", g.soulPower)
	}

	// Second respawn spends the last point.
	g.killPlayer(1)
	g.respawnPlayer(1)
	if g.soulPower != 0 {
		t.Errorf("soulPower = %d, want 0 after second respawn", g.soulPower)
	}

	// Third respawn is refused: no Soul Power left.
	g.killPlayer(1)
	g.respawnPlayer(1)
	if p.hp != 0 {
		t.Errorf("cultist out of Soul Power must stay dead, hp = %d", p.hp)
	}
	if !p.isSpectator {
		t.Error("cultist out of Soul Power should become a spectator")
	}
	if !hasRespawnDenied(client, respawnDeniedNoSoulPower) {
		t.Error("expected RespawnDeniedEvent(noSoulPower)")
	}
}

func TestJoinAfterBossIsSpectator(t *testing.T) {
	g, _ := newTestGame()
	g.demonWasSpawned = true

	joiner := newFakeClient(7)
	g.OnClientJoined(joiner)

	p, ok := g.players[7]
	if !ok {
		t.Fatal("joiner should be added to the game")
	}
	if !p.isSpectator {
		t.Error("a client joining after the boss is revealed must be a spectator")
	}
	if p.hp != 0 {
		t.Errorf("spectator joiner hp = %d, want 0", p.hp)
	}

	var sawGameData, spectatorFlag bool
	for _, e := range joiner.sentEvents {
		if jd, ok := e.(JoinToStartedGameEvent); ok {
			sawGameData = true
			if v, _ := jd.GameData["isSpectator"].(bool); v {
				spectatorFlag = true
			}
		}
	}
	if !sawGameData {
		t.Error("spectator should still receive the battlemap (JoinToStartedGameEvent)")
	}
	if !spectatorFlag {
		t.Error("initial game data should mark the joiner as a spectator")
	}
}

func TestJoinBeforeBossPlaysNormally(t *testing.T) {
	g, _ := newTestGame()

	joiner := newFakeClient(8)
	g.OnClientJoined(joiner)

	p, ok := g.players[8]
	if !ok {
		t.Fatal("joiner should be added to the game")
	}
	if p.isSpectator {
		t.Error("a client joining before the boss should not be a spectator")
	}
	if p.hp <= 0 {
		t.Errorf("joiner before boss should be alive, hp = %d", p.hp)
	}
}

func hasRespawnDenied(c *fakeClient, reason string) bool {
	for _, e := range c.sentEvents {
		if rd, ok := e.(RespawnDeniedEvent); ok && rd.Reason == reason {
			return true
		}
	}
	return false
}

func TestMaxCultistsAllowed(t *testing.T) {
	cases := []struct {
		players int
		want    int
	}{
		{1, 0},
		{2, 0},
		{3, 1},
		{5, 1},
		{6, 2},
	}
	for _, c := range cases {
		g, _ := newTestGame()
		for i := 0; i < c.players; i++ {
			addTestPlayer(g, uint64(i+1), ClassKnight)
		}
		if got := g.maxCultistsAllowedUnsafe(); got != c.want {
			t.Errorf("maxCultistsAllowed(%d players) = %d, want %d", c.players, got, c.want)
		}
	}
}
