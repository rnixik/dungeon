package game

// TrapState represents the current state of a trap in its FSM
type TrapState string

const (
	TrapStateDisabled TrapState = "disabled"
	TrapStateArmed    TrapState = "armed"
	TrapStateActive   TrapState = "active"
	TrapStateCooldown TrapState = "cooldown"
)

// TrapType defines the type of trap
type TrapType string

const (
	TrapTypeSpikes TrapType = "spikes"
	TrapTypeArrows TrapType = "arrows"
	TrapTypeFire   TrapType = "fire"
	TrapTypePoison TrapType = "poison"
)

// ActivatorType defines how a trap is activated
type ActivatorType string

const (
	ActivatorTimer  ActivatorType = "timer"  // Always on, loops with period
	ActivatorLink   ActivatorType = "link"   // Triggered by another object
	ActivatorToggle ActivatorType = "toggle" // Timer on/off
)

// TrapActivator defines the activation source for a trap
type TrapActivator struct {
	Type   ActivatorType
	Period float64 // For timer activators, in seconds
	Phase  float64 // Phase offset for timer activators
	LinkID string  // For link activators, the trigger object ID
}

// TrapParams holds the configuration for a trap
type TrapParams struct {
	ActivePercent   float64 // Percent of period for active phase (rising animation + damage), 0-100
	CooldownPercent float64 // Percent of period for cooldown phase (retracting), 0-100
	Damage          int     // Damage dealt per hit
	X               int     // Position X (tile coordinate)
	Y               int     // Position Y (tile coordinate)
}

// Trap represents a single trap instance with FSM
type Trap struct {
	ID                 string
	Type               TrapType
	State              TrapState
	Params             TrapParams
	Activator          TrapActivator
	StateTimer         float64 // Time remaining in current state
	LoopTimer          float64 // For timer activators
	LastDamagedPlayers map[uint64]bool // Players damaged in current activation cycle
}

// NewTrap creates a new trap instance
func NewTrap(id string, trapType TrapType, params TrapParams, activator TrapActivator) *Trap {
	// Normalize phase to be within period range
	phase := activator.Phase
	if activator.Period > 0 {
		// Use modulo to wrap phase within 0..period range
		for phase >= activator.Period {
			phase -= activator.Period
		}
		for phase < 0 {
			phase += activator.Period
		}
	}
	
	return &Trap{
		ID:         id,
		Type:       trapType,
		State:      TrapStateArmed,
		Params:     params,
		Activator:  activator,
		StateTimer: 0,
		LoopTimer:  phase, // Start with normalized phase offset
		LastDamagedPlayers: make(map[uint64]bool),
	}
}

// Tick updates the trap state based on elapsed time
func (t *Trap) Tick(deltaTime float64) (stateChanged bool, newState TrapState) {
	// Update timer activators
	if t.Activator.Type == ActivatorTimer && t.State == TrapStateArmed {
		t.LoopTimer += deltaTime
		if t.LoopTimer >= t.Activator.Period {
			t.LoopTimer = 0
			t.Activate()
			return true, t.State
		}
	}

	// Update state timer
	if t.StateTimer > 0 {
		t.StateTimer -= deltaTime
		if t.StateTimer <= 0 {
			return t.transitionToNextState()
		}
	}

	return false, t.State
}

// Activate triggers the trap (called by activators or triggers)
func (t *Trap) Activate() {
	if t.State == TrapStateArmed {
		t.State = TrapStateActive
		// Calculate active time from percent of period
		t.StateTimer = t.Activator.Period * (t.Params.ActivePercent / 100.0)
	}
}

// transitionToNextState moves the trap to the next state in the FSM
func (t *Trap) transitionToNextState() (stateChanged bool, newState TrapState) {
	oldState := t.State

	switch t.State {
	case TrapStateActive:
		t.State = TrapStateCooldown
		// Calculate cooldown time from percent of period
		t.StateTimer = t.Activator.Period * (t.Params.CooldownPercent / 100.0)
	case TrapStateCooldown:
		t.State = TrapStateArmed
		t.StateTimer = 0
		// Clear damaged players list when entering armed state (ready for next cycle)
		t.LastDamagedPlayers = make(map[uint64]bool)
	default:
		return false, t.State
	}

	return oldState != t.State, t.State
}

// IsActive returns true if the trap is in the active state (can deal damage)
// Damage is dealt during entire Active state (including rising animation)
func (t *Trap) IsActive() bool {
	return t.State == TrapStateActive
}

// GetCurrentFrame returns the appropriate animation frame based on state
// Frame mapping:
// 0: Active peak (fully extended, damaging)
// 1-4: Cooldown (retracting)
// 5: Armed (hidden)
// 7-11: Active start (rising animation, plays once)
func (t *Trap) GetCurrentFrame() int {
	switch t.State {
	case TrapStateDisabled, TrapStateArmed:
		return 5 // Hidden
		
	case TrapStateActive:
		// Active phase: rising animation (7-11) then extended (0)
		if t.Params.ActivePercent > 0 {
			activeTime := t.Activator.Period * (t.Params.ActivePercent / 100.0)
			progress := 1.0 - (t.StateTimer / activeTime)
			
			// Rising animation plays in first 40% of active time
			if progress < 0.4 {
				// Map 0-0.4 progress to frames 7-11
				frameIndex := int((progress / 0.4) * 5)
				if frameIndex > 4 {
					frameIndex = 4
				}
				return 7 + frameIndex
			}
		}
		return 0 // Fully extended for remaining time
		
	case TrapStateCooldown:
		// Retracting animation: frames 1-4
		if t.Params.CooldownPercent > 0 {
			cooldownTime := t.Activator.Period * (t.Params.CooldownPercent / 100.0)
			progress := 1.0 - (t.StateTimer / cooldownTime)
			frameIndex := int(progress * 4)
			if frameIndex > 3 {
				frameIndex = 3
			}
			return 1 + frameIndex
		}
		return 4
		
	default:
		return 5
	}
}
