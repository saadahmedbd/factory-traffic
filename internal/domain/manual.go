package domain

import (
	"fmt"
	"time"
)

// ApplyManualHold puts the junction into manual override for a target phase and duration.
func ApplyManualHold(state *JunctionState, targetPhase PhaseID, duration time.Duration, requestedBy string, now time.Time) error {
	if _, ok := state.Config.Phases[targetPhase]; !ok && targetPhase != PhaseAllRed {
		return fmt.Errorf("unknown phase: %s", targetPhase)
	}

	maxDuration := state.Config.ManualHoldMaxDuration
	if duration <= 0 || duration > maxDuration {
		duration = maxDuration
	}

	state.Mode = ModeManual
	state.ManualHold = ManualHold{
		Active:      true,
		Phase:       targetPhase,
		StartedAt:   now,
		ExpireAt:    now.Add(duration),
		RequestedBy: requestedBy,
	}

	return nil
}

// ReleaseManualHold terminates the manual override and reverts to Auto mode.
func ReleaseManualHold(state *JunctionState) {
	state.ManualHold = ManualHold{Active: false}
	if state.Mode == ModeManual {
		state.Mode = ModeAuto
	}
}

// CheckManualExpiry verifies if manual hold duration has elapsed, auto-reverting if needed.
func CheckManualExpiry(state *JunctionState, now time.Time) bool {
	if !state.ManualHold.Active {
		return false
	}

	if now.After(state.ManualHold.ExpireAt) || now.Equal(state.ManualHold.ExpireAt) {
		ReleaseManualHold(state)
		return true
	}

	return false
}
