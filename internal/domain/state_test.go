package domain

import (
	"testing"
	"time"
)

func TestNewJunctionState(t *testing.T) {
	now := time.Now()
	cfg := DefaultConfig()
	state := NewJunctionState("junction-1", cfg, now)

	if state.JunctionID != "junction-1" {
		t.Errorf("expected junction-1, got %s", state.JunctionID)
	}
	if state.Mode != ModeAuto {
		t.Errorf("expected ModeAuto, got %s", state.Mode)
	}
	if state.CurrentPhase != PhaseNorthSouth {
		t.Errorf("expected PhaseNorthSouth, got %s", state.CurrentPhase)
	}
	if state.Signals[DirectionNorth] != SignalGreen || state.Signals[DirectionSouth] != SignalGreen {
		t.Errorf("expected North/South to start green")
	}
	if state.Signals[DirectionEast] != SignalRed || state.Signals[DirectionWest] != SignalRed {
		t.Errorf("expected East/West to start red")
	}
}

func TestJunctionStateClone(t *testing.T) {
	now := time.Now()
	cfg := DefaultConfig()
	state := NewJunctionState("junction-1", cfg, now)

	state.Queues[DirectionNorth] = append(state.Queues[DirectionNorth], Vehicle{
		ID:        "v1",
		Type:      VehicleAGV,
		Direction: DirectionNorth,
		ArrivedAt: now,
	})

	clone := state.Clone()
	clone.Queues[DirectionNorth][0].ID = "v-modified"
	clone.Signals[DirectionNorth] = SignalYellow

	if state.Queues[DirectionNorth][0].ID == "v-modified" {
		t.Errorf("modifying clone affected original state queue")
	}
	if state.Signals[DirectionNorth] == SignalYellow {
		t.Errorf("modifying clone signals affected original state signals")
	}
}
