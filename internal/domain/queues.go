package domain

import (
	"sync"
	"time"
)

// DedupeTracker prevents duplicate sensor, RFID triggers, or repeated event_id submissions.
type DedupeTracker struct {
	mu          sync.Mutex
	history     map[string]time.Time
	eventTokens map[string]time.Time
}

// NewDedupeTracker creates a deduplicator instance.
func NewDedupeTracker() *DedupeTracker {
	return &DedupeTracker{
		history:     make(map[string]time.Time),
		eventTokens: make(map[string]time.Time),
	}
}

// IsDuplicate returns true if the vehicleID was seen within the window.
// If not duplicate, it records the observation.
func (dt *DedupeTracker) IsDuplicate(vehicleID string, now time.Time, window time.Duration) bool {
	dt.mu.Lock()
	defer dt.mu.Unlock()

	// Prune old entries
	for id, t := range dt.history {
		if now.Sub(t) > window*2 {
			delete(dt.history, id)
		}
	}

	if lastSeen, found := dt.history[vehicleID]; found {
		if now.Sub(lastSeen) < window {
			return true
		}
	}

	dt.history[vehicleID] = now
	return false
}

// IsEventProcessed returns true if the eventID has already been processed (idempotency).
func (dt *DedupeTracker) IsEventProcessed(eventID string) bool {
	if eventID == "" {
		return false
	}
	dt.mu.Lock()
	defer dt.mu.Unlock()

	if _, found := dt.eventTokens[eventID]; found {
		return true
	}
	dt.eventTokens[eventID] = time.Now()
	return false
}

// EnqueueVehicle adds a vehicle to the appropriate direction queue in state.
func EnqueueVehicle(state *JunctionState, v Vehicle) {
	if state.Queues == nil {
		state.Queues = make(map[Direction][]Vehicle)
	}

	// Avoid adding duplicate within the same queue
	existing := state.Queues[v.Direction]
	for _, item := range existing {
		if item.ID == v.ID {
			return
		}
	}

	state.Queues[v.Direction] = append(existing, v)
}

// DequeueVehicle removes a vehicle by ID from the specified direction.
func DequeueVehicle(state *JunctionState, dir Direction, vehicleID string) (Vehicle, bool) {
	q := state.Queues[dir]
	for i, v := range q {
		if v.ID == vehicleID {
			state.Queues[dir] = append(q[:i], q[i+1:]...)
			return v, true
		}
	}
	return Vehicle{}, false
}

// PopNextVehicle removes and returns the first vehicle waiting in the specified direction.
func PopNextVehicle(state *JunctionState, dir Direction) (Vehicle, bool) {
	q := state.Queues[dir]
	if len(q) == 0 {
		return Vehicle{}, false
	}
	first := q[0]
	state.Queues[dir] = q[1:]
	return first, true
}

// QueueWeight calculates the total weighted score of vehicles waiting in a direction.
func QueueWeight(q []Vehicle, cfg Config) int {
	total := 0
	for _, v := range q {
		total += cfg.WeightFor(v.Type)
	}
	return total
}

// OldestVehicleWaitTime returns the duration the first vehicle in the queue has been waiting.
func OldestVehicleWaitTime(q []Vehicle, now time.Time) time.Duration {
	if len(q) == 0 {
		return 0
	}
	wait := now.Sub(q[0].ArrivedAt)
	if wait < 0 {
		return 0
	}
	return wait
}
