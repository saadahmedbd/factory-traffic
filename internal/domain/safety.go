package domain

import (
	"fmt"
)

// ErrSafetyViolation indicates a fatal safety invariant breach.
type ErrSafetyViolation struct {
	Rule    string
	Details string
}

func (e ErrSafetyViolation) Error() string {
	return fmt.Sprintf("SAFETY VIOLATION [%s]: %s", e.Rule, e.Details)
}

// AssertSafe checks all safety invariants of a junction state.
func AssertSafe(state JunctionState) error {
	signals := state.Signals

	// Invariant 1: Conflicting Directions Mutual Exclusion
	// North/South and East/West must never both have non-Red (Green/Yellow) concurrently
	nsActive := (signals[DirectionNorth] == SignalGreen || signals[DirectionNorth] == SignalYellow ||
		signals[DirectionSouth] == SignalGreen || signals[DirectionSouth] == SignalYellow)

	ewActive := (signals[DirectionEast] == SignalGreen || signals[DirectionEast] == SignalYellow ||
		signals[DirectionWest] == SignalGreen || signals[DirectionWest] == SignalYellow)

	if nsActive && ewActive {
		return ErrSafetyViolation{
			Rule:    "MUTUAL_EXCLUSION",
			Details: fmt.Sprintf("Conflicting movements active simultaneously: North=%s, South=%s, East=%s, West=%s",
				signals[DirectionNorth], signals[DirectionSouth], signals[DirectionEast], signals[DirectionWest]),
		}
	}

	// Invariant 2: Current phase must match allowed directions if Green
	if state.CurrentSignal == SignalGreen && !state.InTransition {
		phaseDef, exists := state.Config.Phases[state.CurrentPhase]
		if !exists {
			return ErrSafetyViolation{
				Rule:    "INVALID_PHASE",
				Details: fmt.Sprintf("Active phase '%s' is not defined in configuration", state.CurrentPhase),
			}
		}

		allowedMap := make(map[Direction]bool)
		for _, d := range phaseDef.AllowedDirections {
			allowedMap[d] = true
		}

		for dir, sig := range signals {
			if sig == SignalGreen && !allowedMap[dir] {
				return ErrSafetyViolation{
					Rule:    "UNAUTHORIZED_GREEN",
					Details: fmt.Sprintf("Direction %s has GREEN signal but is not allowed in phase %s", dir, state.CurrentPhase),
				}
			}
		}
	}

	return nil
}

// SafeSignalsForPhase returns the valid signal map for a given phase and signal state.
func SafeSignalsForPhase(phaseID PhaseID, sig Signal, cfg Config) map[Direction]Signal {
	res := make(map[Direction]Signal)
	for _, d := range AllDirections {
		res[d] = SignalRed
	}

	phaseDef, ok := cfg.Phases[phaseID]
	if !ok || phaseID == PhaseAllRed {
		return res
	}

	for _, d := range phaseDef.AllowedDirections {
		res[d] = sig
	}
	return res
}
