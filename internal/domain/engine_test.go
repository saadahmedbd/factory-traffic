package domain

import (
	"testing"
	"time"
)

func TestEngineDecideVehicleArrivalAndSwitch(t *testing.T) {
	now := time.Now()
	cfg := DefaultConfig()
	cfg.MinGreenDuration = 5 * time.Second
	cfg.YellowDuration = 3 * time.Second
	cfg.AllRedDuration = 2 * time.Second

	// Start at PhaseNorthSouth, satisfied min green
	state := NewJunctionState("j1", cfg, now.Add(-6*time.Second))
	state.CurrentPhase = PhaseNorthSouth

	// East arrival
	vEast := Vehicle{
		ID:        "AGV-1",
		Type:      VehicleAGV,
		Direction: DirectionEast,
		ArrivedAt: now,
	}

	// 1. Vehicle Arrival Input
	s1, eff1, err := Decide(state, EngineInput{
		Type:    InputTypeVehicleArrival,
		Vehicle: &vEast,
	}, now)
	if err != nil {
		t.Fatalf("unexpected error on vehicle arrival: %v", err)
	}

	if len(s1.Queues[DirectionEast]) != 1 {
		t.Fatalf("expected vehicle in East queue")
	}

	// Because min green elapsed (6s > 5s) and current NS has no vehicles while East has 1 AGV,
	// Decide should initiate transition to PhaseEastWest by going YELLOW on North/South!
	if !s1.InTransition {
		t.Fatalf("expected junction to enter transition")
	}
	if s1.TransitionStep != StepYellowClear {
		t.Errorf("expected StepYellowClear, got %s", s1.TransitionStep)
	}
	if s1.Signals[DirectionNorth] != SignalYellow || s1.Signals[DirectionSouth] != SignalYellow {
		t.Errorf("expected NS signals to turn Yellow during clear step")
	}

	// 2. Yellow duration elapses -> advance to AllRed
	yellowExpiredTime := now.Add(cfg.YellowDuration + 100*time.Millisecond)
	s2, eff2, err := Decide(s1, EngineInput{Type: InputTypeTick}, yellowExpiredTime)
	if err != nil {
		t.Fatalf("error advancing through yellow: %v", err)
	}
	if s2.TransitionStep != StepAllRedClear {
		t.Errorf("expected StepAllRedClear, got %s", s2.TransitionStep)
	}
	for dir, sig := range s2.Signals {
		if sig != SignalRed {
			t.Errorf("expected %s to be RED during all-red clear, got %s", dir, sig)
		}
	}

	// 3. AllRed duration elapses -> activate East-West GREEN
	allRedExpiredTime := yellowExpiredTime.Add(cfg.AllRedDuration + 100*time.Millisecond)
	s3, eff3, err := Decide(s2, EngineInput{Type: InputTypeTick}, allRedExpiredTime)
	if err != nil {
		t.Fatalf("error finishing transition: %v", err)
	}
	if s3.InTransition {
		t.Errorf("expected transition to be finished")
	}
	if s3.CurrentPhase != PhaseEastWest {
		t.Errorf("expected current phase to be PhaseEastWest, got %s", s3.CurrentPhase)
	}
	if s3.Signals[DirectionEast] != SignalGreen || s3.Signals[DirectionWest] != SignalGreen {
		t.Errorf("expected East/West signals to be GREEN")
	}
	if s3.Signals[DirectionNorth] != SignalRed || s3.Signals[DirectionSouth] != SignalRed {
		t.Errorf("expected North/South signals to be RED")
	}

	_ = eff1
	_ = eff2
	_ = eff3
}

func TestEngineEmergencyPreemption(t *testing.T) {
	now := time.Now()
	cfg := DefaultConfig()
	cfg.MinGreenDuration = 10 * time.Second
	cfg.YellowDuration = 2 * time.Second

	// Phase just started 1 second ago (min green NOT satisfied yet)
	state := NewJunctionState("j1", cfg, now.Add(-1*time.Second))

	emVehicle := Vehicle{
		ID:        "FIRE-TRUCK-1",
		Type:      VehicleEmergency,
		Direction: DirectionWest,
		ArrivedAt: now,
	}

	s1, _, err := Decide(state, EngineInput{
		Type:    InputTypeEmergencyPreempt,
		Vehicle: &emVehicle,
	}, now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if s1.Mode != ModeEmergency {
		t.Errorf("expected ModeEmergency, got %s", s1.Mode)
	}
	if !s1.InTransition {
		t.Errorf("emergency preemption should override min green and trigger safe transition")
	}
	if s1.TargetPhase != PhaseEastWest {
		t.Errorf("expected target PhaseEastWest, got %s", s1.TargetPhase)
	}
}

func TestEngineDegradedModeOnAckFailure(t *testing.T) {
	now := time.Now()
	cfg := DefaultConfig()
	state := NewJunctionState("j1", cfg, now)

	cmd := buildCommand(&state, CommandSetSignals, now)
	state.PendingCommand = &cmd

	// Hardware sends NACK (rejected)
	s1, _, err := Decide(state, EngineInput{
		Type: InputTypeControllerAck,
		Ack: &ControllerAck{
			CommandID:    cmd.ID,
			Success:      false,
			ErrorMessage: "Hardware coil blown",
			Timestamp:    now,
		},
	}, now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if s1.Mode != ModeDegraded {
		t.Errorf("expected ModeDegraded on controller failure, got %s", s1.Mode)
	}
	for dir, sig := range s1.Signals {
		if sig != SignalYellow {
			t.Errorf("expected direction %s to be Yellow in degraded mode, got %s", dir, sig)
		}
	}
}
