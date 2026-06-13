package game

import "testing"

func newTimerTrap() *Trap {
	return NewTrap("t1", TrapTypeSpikes, TrapParams{
		ActivePercent:   30,
		CooldownPercent: 20,
		Damage:          25,
	}, TrapActivator{
		Type:   ActivatorTimer,
		Period: 2.0,
	})
}

func TestNewTrapStartsArmed(t *testing.T) {
	trap := newTimerTrap()
	if trap.State != TrapStateArmed {
		t.Fatalf("expected new trap to be armed, got %q", trap.State)
	}
	if trap.LastDamagedPlayers == nil || trap.LastDamagedMonsters == nil {
		t.Fatal("expected damaged-tracking maps to be initialized")
	}
}

func TestNewTrapNormalizesPhase(t *testing.T) {
	tests := []struct {
		name     string
		phase    float64
		period   float64
		wantLoop float64
	}{
		{"phase within range", 1.5, 2.0, 1.5},
		{"phase above period wraps down", 5.5, 2.0, 1.5},
		{"negative phase wraps up", -0.5, 2.0, 1.5},
		{"zero period leaves phase untouched", 3.0, 0, 3.0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			trap := NewTrap("t", TrapTypeFire, TrapParams{}, TrapActivator{
				Type:   ActivatorTimer,
				Period: tc.period,
				Phase:  tc.phase,
			})
			if trap.LoopTimer != tc.wantLoop {
				t.Errorf("LoopTimer = %v, want %v", trap.LoopTimer, tc.wantLoop)
			}
		})
	}
}

func TestTrapTimerActivatesAfterPeriod(t *testing.T) {
	trap := newTimerTrap()

	// Not yet a full period -> stays armed.
	if changed, _ := trap.Tick(1.0); changed {
		t.Fatal("trap should not change state before a full period elapses")
	}
	if trap.State != TrapStateArmed {
		t.Fatalf("expected armed, got %q", trap.State)
	}

	// Crossing the period triggers activation.
	changed, state := trap.Tick(1.0)
	if !changed || state != TrapStateActive {
		t.Fatalf("expected activation to TrapStateActive, got changed=%v state=%q", changed, state)
	}
	// Active time = 30% of 2.0s period = 0.6s.
	if trap.StateTimer != 0.6 {
		t.Errorf("active StateTimer = %v, want 0.6", trap.StateTimer)
	}
}

func TestTrapFullCycle(t *testing.T) {
	trap := newTimerTrap()

	// Arm -> Active.
	trap.Tick(2.0)
	if trap.State != TrapStateActive {
		t.Fatalf("expected active after period, got %q", trap.State)
	}
	if !trap.IsActive() {
		t.Fatal("IsActive should be true while active")
	}

	// Active -> Cooldown (active phase = 0.6s).
	_, state := trap.Tick(0.6)
	if state != TrapStateCooldown {
		t.Fatalf("expected cooldown, got %q", state)
	}
	// Cooldown time = 20% of 2.0s = 0.4s.
	if trap.StateTimer != 0.4 {
		t.Errorf("cooldown StateTimer = %v, want 0.4", trap.StateTimer)
	}

	// Cooldown -> Armed.
	_, state = trap.Tick(0.4)
	if state != TrapStateArmed {
		t.Fatalf("expected armed after cooldown, got %q", state)
	}
	if trap.IsActive() {
		t.Fatal("IsActive should be false while armed")
	}
}

func TestTrapClearsDamagedListsOnRearm(t *testing.T) {
	trap := newTimerTrap()
	trap.Tick(2.0) // -> active
	trap.Tick(0.6) // -> cooldown
	trap.LastDamagedPlayers[42] = true
	trap.LastDamagedMonsters[7] = true

	trap.Tick(0.4) // -> armed, should clear

	if len(trap.LastDamagedPlayers) != 0 || len(trap.LastDamagedMonsters) != 0 {
		t.Errorf("expected damaged lists cleared on rearm, got players=%v monsters=%v",
			trap.LastDamagedPlayers, trap.LastDamagedMonsters)
	}
}

func TestTrapActivateOnlyFromArmed(t *testing.T) {
	trap := newTimerTrap()
	trap.State = TrapStateCooldown
	trap.Activate()
	if trap.State != TrapStateCooldown {
		t.Errorf("Activate should be a no-op outside armed state, got %q", trap.State)
	}
}

func TestTrapGetCurrentFrame(t *testing.T) {
	trap := newTimerTrap()

	if got := trap.GetCurrentFrame(); got != 5 {
		t.Errorf("armed frame = %d, want 5 (hidden)", got)
	}

	trap.Tick(2.0) // -> active, StateTimer = 0.6, progress 0 -> rising frame 7
	if got := trap.GetCurrentFrame(); got != 7 {
		t.Errorf("active start frame = %d, want 7", got)
	}

	// Advance past the rising window (first 40% of 0.6s = 0.24s) -> fully extended frame 0.
	trap.Tick(0.3)
	if got := trap.GetCurrentFrame(); got != 0 {
		t.Errorf("active peak frame = %d, want 0", got)
	}

	// Move to cooldown; early cooldown -> frame 1.
	trap.Tick(0.3) // exhausts active (0.6 total) -> cooldown StateTimer 0.4
	if trap.State != TrapStateCooldown {
		t.Fatalf("expected cooldown, got %q", trap.State)
	}
	if got := trap.GetCurrentFrame(); got != 1 {
		t.Errorf("cooldown start frame = %d, want 1", got)
	}
}
