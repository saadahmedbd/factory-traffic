package domain

import "time"

// EffectType categorizes domain output side-effects.
type EffectType string

const (
	EffectTypeSendCommand   EffectType = "SEND_COMMAND"
	EffectTypeAudit         EffectType = "AUDIT"
	EffectTypePersist       EffectType = "PERSIST"
	EffectTypeScheduleTimer EffectType = "SCHEDULE_TIMER"
)

// AuditEvent represents an immutable audit record of operational events.
type AuditEvent struct {
	JunctionID string                 `json:"junction_id"`
	EventType  string                 `json:"event_type"`
	Phase      PhaseID                `json:"phase"`
	Mode       Mode                   `json:"mode"`
	Details    map[string]interface{} `json:"details"`
	Timestamp  time.Time              `json:"timestamp"`
}

// Effect encapsulates an action that the application layer must execute.
type Effect struct {
	Type          EffectType          `json:"type"`
	Command       *ControllerCommand  `json:"command,omitempty"`
	Audit         *AuditEvent         `json:"audit,omitempty"`
	TimerDuration time.Duration       `json:"timer_duration,omitempty"`
	TimerReason   string              `json:"timer_reason,omitempty"`
}

// NewSendCommandEffect constructs an effect to transmit hardware instructions.
func NewSendCommandEffect(cmd ControllerCommand) Effect {
	return Effect{
		Type:    EffectTypeSendCommand,
		Command: &cmd,
	}
}

// NewAuditEffect creates an audit log effect.
func NewAuditEffect(junctionID, eventType string, phase PhaseID, mode Mode, details map[string]interface{}, now time.Time) Effect {
	return Effect{
		Type: EffectTypeAudit,
		Audit: &AuditEvent{
			JunctionID: junctionID,
			EventType:  eventType,
			Phase:      phase,
			Mode:       mode,
			Details:    details,
			Timestamp:  now,
		},
	}
}

// NewPersistEffect creates an effect to persist state snapshot.
func NewPersistEffect() Effect {
	return Effect{
		Type: EffectTypePersist,
	}
}

// NewScheduleTimerEffect instructs the actor loop to wake up after a duration.
func NewScheduleTimerEffect(d time.Duration, reason string) Effect {
	return Effect{
		Type:          EffectTypeScheduleTimer,
		TimerDuration: d,
		TimerReason:   reason,
	}
}
