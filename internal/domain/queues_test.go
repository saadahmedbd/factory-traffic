package domain

import (
	"testing"
	"time"
)

func TestDedupeTracker(t *testing.T) {
	dt := NewDedupeTracker()
	now := time.Now()
	window := 2 * time.Second

	// First observation
	if dt.IsDuplicate("AGV-101", now, window) {
		t.Errorf("first occurrence should not be duplicate")
	}

	// Immediate repeat
	if !dt.IsDuplicate("AGV-101", now.Add(500*time.Millisecond), window) {
		t.Errorf("immediate repeat within window should be duplicate")
	}

	// After window expiration
	if dt.IsDuplicate("AGV-101", now.Add(3*time.Second), window) {
		t.Errorf("observation after window expiry should not be duplicate")
	}
}

func TestEnqueueDequeueVehicle(t *testing.T) {
	now := time.Now()
	cfg := DefaultConfig()
	state := NewJunctionState("j1", cfg, now)

	v1 := Vehicle{ID: "v1", Type: VehicleForklift, Direction: DirectionNorth, ArrivedAt: now}
	v2 := Vehicle{ID: "v2", Type: VehicleAGV, Direction: DirectionNorth, ArrivedAt: now.Add(1 * time.Second)}

	EnqueueVehicle(&state, v1)
	EnqueueVehicle(&state, v2)
	// Duplicate in queue should not be added
	EnqueueVehicle(&state, v1)

	if len(state.Queues[DirectionNorth]) != 2 {
		t.Fatalf("expected 2 vehicles in queue, got %d", len(state.Queues[DirectionNorth]))
	}

	popped, ok := PopNextVehicle(&state, DirectionNorth)
	if !ok || popped.ID != "v1" {
		t.Errorf("expected v1 to pop first, got %v", popped)
	}

	dequeued, ok := DequeueVehicle(&state, DirectionNorth, "v2")
	if !ok || dequeued.ID != "v2" {
		t.Errorf("expected v2 to dequeue, got %v", dequeued)
	}

	if len(state.Queues[DirectionNorth]) != 0 {
		t.Errorf("expected queue to be empty")
	}
}

func TestQueueWeightAndWait(t *testing.T) {
	now := time.Now()
	cfg := DefaultConfig()
	q := []Vehicle{
		{ID: "v1", Type: VehicleAGV, ArrivedAt: now.Add(-10 * time.Second)},      // weight 5
		{ID: "v2", Type: VehicleForklift, ArrivedAt: now.Add(-5 * time.Second)},  // weight 3
	}

	weight := QueueWeight(q, cfg)
	if weight != 8 {
		t.Errorf("expected weight 8, got %d", weight)
	}

	wait := OldestVehicleWaitTime(q, now)
	if wait < 9*time.Second || wait > 11*time.Second {
		t.Errorf("expected wait approx 10s, got %v", wait)
	}
}
