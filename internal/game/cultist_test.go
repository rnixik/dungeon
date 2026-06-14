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
