package domain

import "time"

// PhaseScore holds diagnostic scoring metrics for each candidate phase.
type PhaseScore struct {
	PhaseID         PhaseID       `json:"phase_id"`
	VehicleCount    int           `json:"vehicle_count"`
	TotalWeight     int           `json:"total_weight"`
	StarvedCount    int           `json:"starved_count"`
	StarvationBonus int           `json:"starvation_bonus"`
	MaxWaitDuration time.Duration `json:"max_wait_duration"`
	FinalScore      int           `json:"final_score"`
}

// CalculatePhaseScore calculates dynamic demand score for a phase.
func CalculatePhaseScore(phaseID PhaseID, state *JunctionState, now time.Time) PhaseScore {
	phaseDef, ok := state.Config.Phases[phaseID]
	if !ok || len(phaseDef.AllowedDirections) == 0 {
		return PhaseScore{PhaseID: phaseID}
	}

	score := PhaseScore{PhaseID: phaseID}

	for _, dir := range phaseDef.AllowedDirections {
		queue := state.Queues[dir]
		score.VehicleCount += len(queue)

		for _, v := range queue {
			score.TotalWeight += state.Config.WeightFor(v.Type)
			wait := now.Sub(v.ArrivedAt)
			if wait > score.MaxWaitDuration {
				score.MaxWaitDuration = wait
			}
			if wait >= state.Config.StarvationThreshold {
				score.StarvedCount++
			}
		}
	}

	score.StarvationBonus = score.StarvedCount * state.Config.StarvationBonus
	// Final score components: weighted vehicle count + starvation bonus + seconds waited bonus
	waitSecondsBonus := int(score.MaxWaitDuration.Seconds())
	score.FinalScore = score.TotalWeight + score.StarvationBonus + waitSecondsBonus

	return score
}

// SelectOptimalPhase determines whether the current phase should continue or switch.
// Returns: (targetPhase, shouldSwitch)
func SelectOptimalPhase(state *JunctionState, now time.Time) (PhaseID, bool) {
	// If emergency active, emergency phase takes absolute precedence
	if HasActiveEmergency(state) {
		if emPhase, ok := EmergencyTargetPhase(state); ok {
			if emPhase != state.CurrentPhase {
				return emPhase, true
			}
			return state.CurrentPhase, false
		}
	}

	// If manual hold active, manual phase takes precedence
	if state.ManualHold.Active {
		if state.ManualHold.Phase != state.CurrentPhase {
			return state.ManualHold.Phase, true
		}
		return state.CurrentPhase, false
	}

	timeInPhase := now.Sub(state.PhaseStartedAt)

	// Invariant: Do not switch if minimum green has not elapsed
	if timeInPhase < state.Config.MinGreenDuration {
		return state.CurrentPhase, false
	}

	nsScore := CalculatePhaseScore(PhaseNorthSouth, state, now)
	ewScore := CalculatePhaseScore(PhaseEastWest, state, now)

	currentIsNS := state.CurrentPhase == PhaseNorthSouth
	opposingScore := ewScore
	currentScore := nsScore
	opposingPhase := PhaseEastWest

	if !currentIsNS {
		opposingScore = nsScore
		currentScore = ewScore
		opposingPhase = PhaseNorthSouth
	}

	// Rule 1: Max green exceeded and opposing corridor has traffic waiting -> force switch
	if timeInPhase >= state.Config.MaxGreenDuration && opposingScore.VehicleCount > 0 {
		return opposingPhase, true
	}

	// Rule 2: Current corridor is completely clear, but opposing corridor has vehicles waiting -> switch immediately
	if currentScore.VehicleCount == 0 && opposingScore.VehicleCount > 0 {
		return opposingPhase, true
	}

	// Rule 3: Starvation trigger - if opposing corridor has starved vehicles -> switch
	if opposingScore.StarvedCount > 0 && currentScore.StarvedCount == 0 {
		return opposingPhase, true
	}

	// Rule 4: Hysteresis - opposing score must exceed current score by a margin to prevent oscillating switches
	const hysteresisMargin = 3
	if opposingScore.FinalScore > currentScore.FinalScore+hysteresisMargin {
		return opposingPhase, true
	}

	return state.CurrentPhase, false
}
