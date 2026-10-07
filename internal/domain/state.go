package domain

import (
	"time"
)

// ManualHold represents an operator override state.
type ManualHold struct {
	Active      bool      `json:"active"`
	Phase       PhaseID   `json:"phase"`
	StartedAt   time.Time `json:"started_at"`
	ExpireAt    time.Time `json:"expire_at"`
	RequestedBy string    `json:"requested_by"`
}

// JunctionState represents the full operational state of a factory junction.
type JunctionState struct {
	JunctionID      string                 `json:"junction_id"`
	Mode            Mode                   `json:"mode"`
	CurrentPhase    PhaseID                `json:"current_phase"`
	CurrentSignal   Signal                 `json:"current_signal"`
	PhaseStartedAt  time.Time              `json:"phase_started_at"`
	InTransition    bool                   `json:"in_transition"`
	TargetPhase     PhaseID                `json:"target_phase"`
	TransitionStep  TransitionStep         `json:"transition_step"`
	StepStartedAt   time.Time              `json:"step_started_at"`
	Queues          map[Direction][]Vehicle `json:"queues"`
	EmergencyQueue  []Vehicle              `json:"emergency_queue"`
	ManualHold      ManualHold             `json:"manual_hold"`
	Signals         map[Direction]Signal   `json:"signals"`
	Config          Config                 `json:"config"`
	Version         int64                  `json:"version"`
	LastUpdated     time.Time              `json:"last_updated"`
	PendingCommand  *ControllerCommand     `json:"pending_command,omitempty"`
	FailSafeReason  string                 `json:"fail_safe_reason,omitempty"`
}

// NewJunctionState initializes a default safe junction state.
func NewJunctionState(junctionID string, cfg Config, now time.Time) JunctionState {
	queues := make(map[Direction][]Vehicle)
	signals := make(map[Direction]Signal)
	for _, d := range AllDirections {
		queues[d] = make([]Vehicle, 0)
		signals[d] = SignalRed
	}

	// Default starting phase: North-South Green
	signals[DirectionNorth] = SignalGreen
	signals[DirectionSouth] = SignalGreen

	return JunctionState{
		JunctionID:     junctionID,
		Mode:           ModeAuto,
		CurrentPhase:   PhaseNorthSouth,
		CurrentSignal:  SignalGreen,
		PhaseStartedAt: now,
		InTransition:   false,
		TargetPhase:    "",
		TransitionStep: StepNone,
		StepStartedAt:  now,
		Queues:         queues,
		EmergencyQueue: make([]Vehicle, 0),
		ManualHold:     ManualHold{Active: false},
		Signals:        signals,
		Config:         cfg,
		Version:        1,
		LastUpdated:    now,
	}
}

// Clone creates a deep copy of JunctionState.
func (s JunctionState) Clone() JunctionState {
	clone := s
	clone.Queues = make(map[Direction][]Vehicle)
	for d, q := range s.Queues {
		qCopy := make([]Vehicle, len(q))
		copy(qCopy, q)
		clone.Queues[d] = qCopy
	}

	clone.EmergencyQueue = make([]Vehicle, len(s.EmergencyQueue))
	copy(clone.EmergencyQueue, s.EmergencyQueue)

	clone.Signals = make(map[Direction]Signal)
	for d, sig := range s.Signals {
		clone.Signals[d] = sig
	}

	if s.PendingCommand != nil {
		cmdCopy := *s.PendingCommand
		clone.PendingCommand = &cmdCopy
	}

	return clone
}
