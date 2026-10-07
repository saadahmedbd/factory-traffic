package domain

import "time"

// CommandType defines the hardware action instructed to the traffic controller.
type CommandType string

const (
	CommandSetSignals  CommandType = "SET_SIGNALS"
	CommandAllRed      CommandType = "ALL_RED"
	CommandYellowFlash CommandType = "YELLOW_FLASH"
)

// CommandStatus tracks acknowledgment state.
type CommandStatus string

const (
	CommandStatusPending CommandStatus = "PENDING"
	CommandStatusAcked   CommandStatus = "ACKED"
	CommandStatusFailed  CommandStatus = "FAILED"
	CommandStatusTimeout CommandStatus = "TIMEOUT"
)

// ControllerCommand models an instruction sent to the physical or simulated signal controller.
type ControllerCommand struct {
	ID        string               `json:"id"`
	JunctionID string              `json:"junction_id"`
	Seq       int64                `json:"seq"`
	Type      CommandType          `json:"type"`
	Signals   map[Direction]Signal `json:"signals"`
	IssuedAt  time.Time            `json:"issued_at"`
	Deadline  time.Time            `json:"deadline"`
	Retries   int                  `json:"retries"`
	Status    CommandStatus        `json:"status"`
}

// ControllerAck models an acknowledgment returned by the signal controller.
type ControllerAck struct {
	CommandID     string    `json:"command_id"`
	JunctionID    string    `json:"junction_id"`
	Seq           int64     `json:"seq"`
	Success       bool      `json:"success"`
	ErrorMessage  string    `json:"error_message,omitempty"`
	Timestamp     time.Time `json:"timestamp"`
}
