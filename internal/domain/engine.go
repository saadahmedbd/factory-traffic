package domain

import (
	"fmt"
	"time"
)

// InputType enumerates incoming engine events.
type InputType string

const (
	InputTypeTick               InputType = "TICK"
	InputTypeVehicleArrival     InputType = "VEHICLE_ARRIVAL"
	InputTypeVehicleCleared     InputType = "VEHICLE_CLEARED"
	InputTypeManualOverride     InputType = "MANUAL_OVERRIDE"
	InputTypeReleaseManual      InputType = "RELEASE_MANUAL"
	InputTypeEmergencyPreempt   InputType = "EMERGENCY_PREEMPT"
	InputTypeEmergencyCleared   InputType = "EMERGENCY_CLEARED"
	InputTypeControllerAck      InputType = "CONTROLLER_ACK"
	InputTypeEnterDegraded      InputType = "ENTER_DEGRADED"
)

// EngineInput conveys external inputs to the pure decision engine.
type EngineInput struct {
	Type             InputType          `json:"type"`
	EventID          string             `json:"event_id,omitempty"`
	SequenceNo       int64              `json:"sequence_no,omitempty"`
	Vehicle          *Vehicle           `json:"vehicle,omitempty"`
	ClearedVehicleID string             `json:"cleared_vehicle_id,omitempty"`
	ClearedDirection Direction          `json:"cleared_direction,omitempty"`
	ManualPhase      PhaseID            `json:"manual_phase,omitempty"`
	ManualDuration   time.Duration      `json:"manual_duration,omitempty"`
	ManualUser       string             `json:"manual_user,omitempty"`
	Ack              *ControllerAck     `json:"ack,omitempty"`
	DegradedReason   string             `json:"degraded_reason,omitempty"`
}

// Decide executes pure state transition logic without side effects.
// It returns the updated JunctionState, a list of domain Effects to perform, and any error.
func Decide(currState JunctionState, input EngineInput, now time.Time) (JunctionState, []Effect, error) {
	state := currState.Clone()
	state.LastUpdated = now
	state.Version++
	var effects []Effect

	// 1. Process specific input events
	switch input.Type {
	case InputTypeEnterDegraded:
		state.Mode = ModeDegraded
		state.FailSafeReason = input.DegradedReason
		state.InTransition = false
		state.CurrentSignal = SignalYellow
		for _, d := range AllDirections {
			state.Signals[d] = SignalYellow
		}
		cmd := buildCommand(&state, CommandYellowFlash, now)
		effects = append(effects,
			NewSendCommandEffect(cmd),
			NewAuditEffect(state.JunctionID, "DEGRADED_MODE_ENTERED", state.CurrentPhase, state.Mode, map[string]interface{}{
				"reason": input.DegradedReason,
			}, now),
			NewPersistEffect(),
		)
		return state, effects, nil

	case InputTypeVehicleArrival:
		if input.Vehicle != nil {
			if input.Vehicle.Type == VehicleEmergency {
				AddEmergencyVehicle(&state, *input.Vehicle, now)
				effects = append(effects, NewAuditEffect(state.JunctionID, "EMERGENCY_ARRIVAL", state.CurrentPhase, state.Mode, map[string]interface{}{
					"vehicle_id": input.Vehicle.ID,
					"direction":  input.Vehicle.Direction,
				}, now))
			} else {
				EnqueueVehicle(&state, *input.Vehicle)
				effects = append(effects, NewAuditEffect(state.JunctionID, "VEHICLE_ARRIVAL", state.CurrentPhase, state.Mode, map[string]interface{}{
					"vehicle_id":   input.Vehicle.ID,
					"vehicle_type": input.Vehicle.Type,
					"direction":    input.Vehicle.Direction,
				}, now))
			}
			effects = append(effects, NewPersistEffect())
		}

	case InputTypeVehicleCleared:
		if input.ClearedVehicleID != "" {
			if _, ok := DequeueVehicle(&state, input.ClearedDirection, input.ClearedVehicleID); ok {
				effects = append(effects,
					NewAuditEffect(state.JunctionID, "VEHICLE_CLEARED", state.CurrentPhase, state.Mode, map[string]interface{}{
						"vehicle_id": input.ClearedVehicleID,
						"direction":  input.ClearedDirection,
					}, now),
					NewPersistEffect(),
				)
			}
		}

	case InputTypeEmergencyPreempt:
		if input.Vehicle != nil {
			AddEmergencyVehicle(&state, *input.Vehicle, now)
			effects = append(effects,
				NewAuditEffect(state.JunctionID, "EMERGENCY_PREEMPTION_TRIGGERED", state.CurrentPhase, state.Mode, map[string]interface{}{
					"vehicle_id": input.Vehicle.ID,
					"direction":  input.Vehicle.Direction,
				}, now),
				NewPersistEffect(),
			)
		}

	case InputTypeEmergencyCleared:
		if input.ClearedVehicleID != "" {
			if ClearEmergencyVehicle(&state, input.ClearedVehicleID) {
				effects = append(effects,
					NewAuditEffect(state.JunctionID, "EMERGENCY_CLEARED", state.CurrentPhase, state.Mode, map[string]interface{}{
						"vehicle_id": input.ClearedVehicleID,
					}, now),
					NewPersistEffect(),
				)
			}
		}

	case InputTypeManualOverride:
		if err := ApplyManualHold(&state, input.ManualPhase, input.ManualDuration, input.ManualUser, now); err != nil {
			return currState, nil, err
		}
		effects = append(effects,
			NewAuditEffect(state.JunctionID, "MANUAL_OVERRIDE_ENABLED", state.ManualHold.Phase, state.Mode, map[string]interface{}{
				"phase":        input.ManualPhase,
				"duration_sec": input.ManualDuration.Seconds(),
				"user":         input.ManualUser,
			}, now),
			NewPersistEffect(),
		)

	case InputTypeReleaseManual:
		ReleaseManualHold(&state)
		effects = append(effects,
			NewAuditEffect(state.JunctionID, "MANUAL_OVERRIDE_RELEASED", state.CurrentPhase, state.Mode, nil, now),
			NewPersistEffect(),
		)

	case InputTypeControllerAck:
		if input.Ack != nil && state.PendingCommand != nil && state.PendingCommand.ID == input.Ack.CommandID {
			if input.Ack.Success {
				state.PendingCommand.Status = CommandStatusAcked
				state.PendingCommand = nil
			} else {
				state.PendingCommand.Status = CommandStatusFailed
				// Handle ACK failure: transition to failsafe
				state.Mode = ModeDegraded
				state.FailSafeReason = fmt.Sprintf("Controller rejected command %s: %s", input.Ack.CommandID, input.Ack.ErrorMessage)
				for _, d := range AllDirections {
					state.Signals[d] = SignalYellow
				}
				cmd := buildCommand(&state, CommandYellowFlash, now)
				effects = append(effects,
					NewSendCommandEffect(cmd),
					NewAuditEffect(state.JunctionID, "CONTROLLER_ACK_FAILED", state.CurrentPhase, state.Mode, map[string]interface{}{
						"error": input.Ack.ErrorMessage,
					}, now),
					NewPersistEffect(),
				)
				return state, effects, nil
			}
			effects = append(effects, NewPersistEffect())
		}

	case InputTypeTick:
		// Periodic maintenance
		if CheckManualExpiry(&state, now) {
			effects = append(effects,
				NewAuditEffect(state.JunctionID, "MANUAL_OVERRIDE_EXPIRED", state.CurrentPhase, state.Mode, nil, now),
				NewPersistEffect(),
			)
		}
		if PruneExpiredEmergencies(&state, now) {
			effects = append(effects,
				NewAuditEffect(state.JunctionID, "EMERGENCY_EXPIRED", state.CurrentPhase, state.Mode, nil, now),
				NewPersistEffect(),
			)
		}
	}

	// 2. Controller ACK timeout surveillance
	if state.PendingCommand != nil && state.PendingCommand.Status == CommandStatusPending {
		if now.After(state.PendingCommand.Deadline) {
			if state.PendingCommand.Retries < state.Config.MaxAckRetries {
				state.PendingCommand.Retries++
				state.PendingCommand.Deadline = now.Add(state.Config.AckTimeout)
				effects = append(effects,
					NewSendCommandEffect(*state.PendingCommand),
					NewAuditEffect(state.JunctionID, "CONTROLLER_COMMAND_RETRY", state.CurrentPhase, state.Mode, map[string]interface{}{
						"command_id": state.PendingCommand.ID,
						"retry":      state.PendingCommand.Retries,
					}, now),
				)
			} else {
				// Hardware unresponsive -> enter failsafe Degraded mode
				state.Mode = ModeDegraded
				state.FailSafeReason = fmt.Sprintf("Controller failed to ACK command %s after %d retries", state.PendingCommand.ID, state.Config.MaxAckRetries)
				state.PendingCommand = nil
				for _, d := range AllDirections {
					state.Signals[d] = SignalYellow
				}
				cmd := buildCommand(&state, CommandYellowFlash, now)
				effects = append(effects,
					NewSendCommandEffect(cmd),
					NewAuditEffect(state.JunctionID, "CONTROLLER_TIMEOUT_FAILSAFE", state.CurrentPhase, state.Mode, map[string]interface{}{
						"reason": state.FailSafeReason,
					}, now),
					NewPersistEffect(),
				)
				return state, effects, nil
			}
		}
	}

	// If in Degraded mode, skip active phase sequencing
	if state.Mode == ModeDegraded {
		return state, effects, nil
	}

	// 3. Phase Transition State Machine Progression
	if state.InTransition {
		switch state.TransitionStep {
		case StepYellowClear:
			if now.Sub(state.StepStartedAt) >= state.Config.YellowDuration {
				// Advance to StepAllRedClear
				state.TransitionStep = StepAllRedClear
				state.StepStartedAt = now
				state.CurrentSignal = SignalRed
				for _, d := range AllDirections {
					state.Signals[d] = SignalRed
				}
				cmd := buildCommand(&state, CommandAllRed, now)
				effects = append(effects,
					NewSendCommandEffect(cmd),
					NewAuditEffect(state.JunctionID, "TRANSITION_ALL_RED", state.CurrentPhase, state.Mode, map[string]interface{}{
						"target_phase": state.TargetPhase,
					}, now),
					NewPersistEffect(),
					NewScheduleTimerEffect(state.Config.AllRedDuration, "ALL_RED_EXPIRED"),
				)
			}

		case StepAllRedClear:
			if now.Sub(state.StepStartedAt) >= state.Config.AllRedDuration {
				// Transition complete: switch to TargetPhase and turn GREEN
				state.CurrentPhase = state.TargetPhase
				state.CurrentSignal = SignalGreen
				state.InTransition = false
				state.TransitionStep = StepNone
				state.PhaseStartedAt = now
				state.StepStartedAt = now
				state.Signals = SafeSignalsForPhase(state.CurrentPhase, SignalGreen, state.Config)

				cmd := buildCommand(&state, CommandSetSignals, now)
				effects = append(effects,
					NewSendCommandEffect(cmd),
					NewAuditEffect(state.JunctionID, "TRANSITION_GREEN_ACTIVATED", state.CurrentPhase, state.Mode, map[string]interface{}{
						"active_phase": state.CurrentPhase,
					}, now),
					NewPersistEffect(),
					NewScheduleTimerEffect(state.Config.MinGreenDuration, "MIN_GREEN_EXPIRED"),
				)
			}
		}

		if err := AssertSafe(state); err != nil {
			return currState, nil, err
		}
		return state, effects, nil
	}

	// 4. Autonomous & Manual Phase Selection
	targetPhase, shouldSwitch := SelectOptimalPhase(&state, now)
	if shouldSwitch && targetPhase != state.CurrentPhase {
		// Begin safe transition: switch active phase to YELLOW first
		state.InTransition = true
		state.TargetPhase = targetPhase
		state.TransitionStep = StepYellowClear
		state.StepStartedAt = now
		state.CurrentSignal = SignalYellow
		state.Signals = SafeSignalsForPhase(state.CurrentPhase, SignalYellow, state.Config)

		cmd := buildCommand(&state, CommandSetSignals, now)
		effects = append(effects,
			NewSendCommandEffect(cmd),
			NewAuditEffect(state.JunctionID, "TRANSITION_YELLOW_INITIATED", state.CurrentPhase, state.Mode, map[string]interface{}{
				"from_phase": state.CurrentPhase,
				"to_phase":   targetPhase,
			}, now),
			NewPersistEffect(),
			NewScheduleTimerEffect(state.Config.YellowDuration, "YELLOW_CLEAR_EXPIRED"),
		)
	}

	// Verify safety invariants before returning
	if err := AssertSafe(state); err != nil {
		return currState, nil, err
	}

	return state, effects, nil
}

func buildCommand(state *JunctionState, cmdType CommandType, now time.Time) ControllerCommand {
	seq := state.Version
	cmd := ControllerCommand{
		ID:         fmt.Sprintf("cmd-%s-%d", state.JunctionID, seq),
		JunctionID: state.JunctionID,
		Seq:        seq,
		Type:       cmdType,
		Signals:    SafeSignalsForPhase(state.CurrentPhase, state.CurrentSignal, state.Config),
		IssuedAt:   now,
		Deadline:   now.Add(state.Config.AckTimeout),
		Retries:    0,
		Status:     CommandStatusPending,
	}

	if cmdType == CommandYellowFlash {
		cmd.Signals = make(map[Direction]Signal)
		for _, d := range AllDirections {
			cmd.Signals[d] = SignalYellow
		}
	} else if cmdType == CommandAllRed {
		cmd.Signals = make(map[Direction]Signal)
		for _, d := range AllDirections {
			cmd.Signals[d] = SignalRed
		}
	} else {
		cmd.Signals = make(map[Direction]Signal)
		for k, v := range state.Signals {
			cmd.Signals[k] = v
		}
	}

	state.PendingCommand = &cmd
	return cmd
}
