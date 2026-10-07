package domain

import (
	"testing"
	"time"
)

func TestSchedulerStarvationBoost(t *testing.T) {
	now := time.Now()
	cfg := DefaultConfig()
	cfg.MinGreenDuration = 5 * time.Second
	cfg.StarvationThreshold = 10 * time.Second
	cfg.StarvationBonus = 20

	state := NewJunctionState("j1", cfg, now.Add(-10*time.Second))
	state.CurrentPhase = PhaseNorthSouth

	// East queue has a vehicle waiting for 15 seconds (starved!)
	state.Queues[DirectionEast] = append(state.Queues[DirectionEast], Vehicle{
		ID:        "v-starved",
		Type:      VehicleNormal,
		Direction: DirectionEast,
		ArrivedAt: now.Add(-15 * time.Second),
	})

	score := CalculatePhaseScore(PhaseEastWest, &state, now)
	if score.StarvedCount != 1 {
		t.Errorf("expected starved count 1, got %d", score.StarvedCount)
	}
	if score.StarvationBonus != 20 {
		t.Errorf("expected starvation bonus 20, got %d", score.StarvationBonus)
	}

	targetPhase, shouldSwitch := SelectOptimalPhase(&state, now)
	if !shouldSwitch || targetPhase != PhaseEastWest {
		t.Errorf("expected starvation to trigger switch to PhaseEastWest, got switch=%v, target=%s", shouldSwitch, targetPhase)
	}
}

func TestSchedulerMinGreenRespected(t *testing.T) {
	now := time.Now()
	cfg := DefaultConfig()
	cfg.MinGreenDuration = 10 * time.Second

	// Phase just started 2 seconds ago
	state := NewJunctionState("j1", cfg, now.Add(-2*time.Second))
	state.CurrentPhase = PhaseNorthSouth

	// Heavy demand on East/West
	state.Queues[DirectionEast] = append(state.Queues[DirectionEast], Vehicle{
		ID:        "heavy-agv",
		Type:      VehicleAGV,
		Direction: DirectionEast,
		ArrivedAt: now,
	})

	_, shouldSwitch := SelectOptimalPhase(&state, now)
	if shouldSwitch {
		t.Errorf("expected switch to be denied because min green has not elapsed")
	}
}
