package domain

import (
	"testing"
	"time"
)

func TestEmergencyPreemptionLifecycle(t *testing.T) {
	now := time.Now()
	cfg := DefaultConfig()
	state := NewJunctionState("j1", cfg, now)

	emVehicle := Vehicle{
		ID:        "AMBULANCE-1",
		Direction: DirectionEast,
		ArrivedAt: now,
	}

	AddEmergencyVehicle(&state, emVehicle, now)

	if state.Mode != ModeEmergency {
		t.Errorf("expected ModeEmergency, got %s", state.Mode)
	}
	if !HasActiveEmergency(&state) {
		t.Errorf("expected active emergency")
	}

	targetPhase, ok := EmergencyTargetPhase(&state)
	if !ok || targetPhase != PhaseEastWest {
		t.Errorf("expected target PhaseEastWest for East emergency vehicle, got %s", targetPhase)
	}

	// Clearing emergency vehicle
	cleared := ClearEmergencyVehicle(&state, "AMBULANCE-1")
	if !cleared {
		t.Errorf("expected emergency vehicle to clear successfully")
	}
	if state.Mode != ModeAuto {
		t.Errorf("expected mode to revert to AUTO after clearing emergency")
	}
}

func TestEmergencyTTLPruning(t *testing.T) {
	now := time.Now()
	cfg := DefaultConfig()
	cfg.EmergencyTTL = 10 * time.Second
	state := NewJunctionState("j1", cfg, now)

	emVehicle := Vehicle{
		ID:        "EM-OLD",
		Direction: DirectionWest,
		ArrivedAt: now,
	}
	AddEmergencyVehicle(&state, emVehicle, now)

	// After 15 seconds, should expire
	future := now.Add(15 * time.Second)
	changed := PruneExpiredEmergencies(&state, future)

	if !changed {
		t.Errorf("expected prune to remove expired emergency vehicle")
	}
	if HasActiveEmergency(&state) {
		t.Errorf("expected no active emergencies after TTL prune")
	}
	if state.Mode != ModeAuto {
		t.Errorf("expected mode to revert to AUTO")
	}
}
