package domain

import (
	"testing"
	"time"
)

func TestManualOverrideAndExpiry(t *testing.T) {
	now := time.Now()
	cfg := DefaultConfig()
	state := NewJunctionState("j1", cfg, now)

	err := ApplyManualHold(&state, PhaseEastWest, 20*time.Second, "supervisor_bob", now)
	if err != nil {
		t.Fatalf("unexpected error applying manual hold: %v", err)
	}

	if state.Mode != ModeManual {
		t.Errorf("expected ModeManual, got %s", state.Mode)
	}
	if !state.ManualHold.Active || state.ManualHold.Phase != PhaseEastWest {
		t.Errorf("manual hold metadata incorrect: %+v", state.ManualHold)
	}

	// Before expiry
	if CheckManualExpiry(&state, now.Add(10*time.Second)) {
		t.Errorf("should not expire before duration has passed")
	}

	// After expiry
	if !CheckManualExpiry(&state, now.Add(21*time.Second)) {
		t.Errorf("expected manual hold to expire after 21s")
	}
	if state.Mode != ModeAuto {
		t.Errorf("expected mode to revert to AUTO upon expiry")
	}
}

func TestManualOverrideRelease(t *testing.T) {
	now := time.Now()
	cfg := DefaultConfig()
	state := NewJunctionState("j1", cfg, now)

	_ = ApplyManualHold(&state, PhaseEastWest, 30*time.Second, "admin", now)
	ReleaseManualHold(&state)

	if state.ManualHold.Active {
		t.Errorf("expected manual hold to be inactive after release")
	}
	if state.Mode != ModeAuto {
		t.Errorf("expected ModeAuto after release")
	}
}
