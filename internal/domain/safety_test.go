package domain

import (
	"testing"
	"time"
)

func TestAssertSafeValidStates(t *testing.T) {
	now := time.Now()
	cfg := DefaultConfig()
	state := NewJunctionState("j1", cfg, now)

	if err := AssertSafe(state); err != nil {
		t.Fatalf("expected initial state to be safe, got: %v", err)
	}
}

func TestAssertSafeMutualExclusionViolation(t *testing.T) {
	now := time.Now()
	cfg := DefaultConfig()
	state := NewJunctionState("j1", cfg, now)

	// Danger: Turn East GREEN while North is GREEN!
	state.Signals[DirectionEast] = SignalGreen

	err := AssertSafe(state)
	if err == nil {
		t.Fatalf("expected safety violation when North and East are both Green")
	}

	if _, ok := err.(ErrSafetyViolation); !ok {
		t.Fatalf("expected ErrSafetyViolation, got %T", err)
	}
}

func TestAssertSafeUnauthorizedGreen(t *testing.T) {
	now := time.Now()
	cfg := DefaultConfig()
	state := NewJunctionState("j1", cfg, now)
	state.CurrentPhase = PhaseNorthSouth
	state.CurrentSignal = SignalGreen
	// If only East is green during PhaseNorthSouth
	state.Signals[DirectionNorth] = SignalRed
	state.Signals[DirectionSouth] = SignalRed
	state.Signals[DirectionEast] = SignalGreen

	err := AssertSafe(state)
	if err == nil {
		t.Fatalf("expected safety violation for unauthorized green direction")
	}
}
