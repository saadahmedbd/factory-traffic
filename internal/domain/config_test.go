package domain

import (
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.MinGreenDuration <= 0 {
		t.Fatalf("expected positive MinGreenDuration, got %v", cfg.MinGreenDuration)
	}
	if cfg.YellowDuration <= 0 {
		t.Fatalf("expected positive YellowDuration, got %v", cfg.YellowDuration)
	}
	if cfg.AllRedDuration <= 0 {
		t.Fatalf("expected positive AllRedDuration, got %v", cfg.AllRedDuration)
	}

	if cfg.WeightFor(VehicleEmergency) <= cfg.WeightFor(VehicleAGV) {
		t.Errorf("emergency vehicle weight must be higher than AGV weight")
	}

	if cfg.WeightFor("UNKNOWN_TYPE") != 1 {
		t.Errorf("expected default weight of 1 for unknown type, got %d", cfg.WeightFor("UNKNOWN_TYPE"))
	}

	if _, ok := cfg.Phases[PhaseNorthSouth]; !ok {
		t.Errorf("expected PhaseNorthSouth to exist in config")
	}
}

func TestConfigTimingsHierarchy(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.MaxGreenDuration <= cfg.MinGreenDuration {
		t.Errorf("MaxGreenDuration (%v) should exceed MinGreenDuration (%v)", cfg.MaxGreenDuration, cfg.MinGreenDuration)
	}
	if cfg.StarvationThreshold < 5*time.Second {
		t.Errorf("StarvationThreshold should be reasonable, got %v", cfg.StarvationThreshold)
	}
}
