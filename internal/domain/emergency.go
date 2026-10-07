package domain

import "time"

// AddEmergencyVehicle queues an emergency vehicle and marks high priority.
func AddEmergencyVehicle(state *JunctionState, v Vehicle, now time.Time) {
	v.Type = VehicleEmergency
	if v.ArrivedAt.IsZero() {
		v.ArrivedAt = now
	}

	// Avoid duplicates
	for _, ev := range state.EmergencyQueue {
		if ev.ID == v.ID {
			return
		}
	}
	state.EmergencyQueue = append(state.EmergencyQueue, v)
	state.Mode = ModeEmergency
}

// PruneExpiredEmergencies removes stale emergency requests older than TTL.
func PruneExpiredEmergencies(state *JunctionState, now time.Time) bool {
	valid := make([]Vehicle, 0, len(state.EmergencyQueue))
	ttl := state.Config.EmergencyTTL
	for _, ev := range state.EmergencyQueue {
		if now.Sub(ev.ArrivedAt) <= ttl {
			valid = append(valid, ev)
		}
	}

	changed := len(valid) != len(state.EmergencyQueue)
	state.EmergencyQueue = valid

	if len(state.EmergencyQueue) == 0 && state.Mode == ModeEmergency {
		state.Mode = ModeAuto
	}
	return changed
}

// HasActiveEmergency checks if any non-expired emergency vehicle is waiting.
func HasActiveEmergency(state *JunctionState) bool {
	return len(state.EmergencyQueue) > 0
}

// EmergencyTargetPhase determines which phase serves the highest priority waiting emergency vehicle.
func EmergencyTargetPhase(state *JunctionState) (PhaseID, bool) {
	if len(state.EmergencyQueue) == 0 {
		return "", false
	}

	first := state.EmergencyQueue[0]
	switch first.Direction {
	case DirectionNorth, DirectionSouth:
		return PhaseNorthSouth, true
	case DirectionEast, DirectionWest:
		return PhaseEastWest, true
	default:
		return "", false
	}
}

// ClearEmergencyVehicle removes an emergency vehicle once cleared through junction.
func ClearEmergencyVehicle(state *JunctionState, vehicleID string) bool {
	for i, ev := range state.EmergencyQueue {
		if ev.ID == vehicleID {
			state.EmergencyQueue = append(state.EmergencyQueue[:i], state.EmergencyQueue[i+1:]...)
			if len(state.EmergencyQueue) == 0 && state.Mode == ModeEmergency {
				state.Mode = ModeAuto
			}
			return true
		}
	}
	return false
}
