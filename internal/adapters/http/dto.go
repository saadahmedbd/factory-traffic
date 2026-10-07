package http

import (
	"factory-traffic/internal/domain"
)

// ApiResponse represents a standard JSON response wrapper.
type ApiResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// SensorEventRequest models Section 10.2 sensor event specification.
type SensorEventRequest struct {
	EventID     string             `json:"event_id"`
	JunctionID  string             `json:"junction_id"`
	Direction   domain.Direction   `json:"direction"`
	EventType   string             `json:"event_type"` // "VEHICLE_ARRIVED" or "VEHICLE_CLEARED"
	VehicleID   string             `json:"vehicle_id"`
	VehicleType domain.VehicleType `json:"vehicle_type,omitempty"`
	SequenceNo  int64              `json:"sequence_no,omitempty"`
	Timestamp   string             `json:"timestamp,omitempty"`
}

// JunctionStatusResponse models Section 10.3 / 14.2 junction status specification.
type JunctionStatusResponse struct {
	JunctionID       string                         `json:"junction_id"`
	Mode             string                         `json:"mode"`
	Phase            string                         `json:"phase"`
	ControllerStatus string                         `json:"controller_status"`
	DesiredSignals   map[domain.Direction]domain.Signal `json:"desired_signals"`
	ActualSignals    map[domain.Direction]domain.Signal `json:"actual_signals"`
	Queues           map[domain.Direction]int              `json:"queues"`
	Vehicles         map[domain.Direction][]domain.Vehicle `json:"vehicles,omitempty"`
	PendingCommand   *domain.ControllerCommand             `json:"pending_command,omitempty"`
	InTransition     bool                                  `json:"in_transition"`
	TransitionStep   domain.TransitionStep                 `json:"transition_step,omitempty"`
	ActiveAlerts     []string                              `json:"active_alerts,omitempty"`
}

// ManualCommandRequest models Section 10.4 manual commands.
type ManualCommandRequest struct {
	Command   string           `json:"command"` // "MANUAL_GREEN_REQUEST" or "RETURN_TO_AUTOMATIC"
	Direction domain.Direction `json:"direction,omitempty"`
	Phase     domain.PhaseID   `json:"phase,omitempty"`
	Duration  int              `json:"duration_seconds,omitempty"`
}

// ControllerEventRequest models Section 10.5 hardware controller acknowledgements.
type ControllerEventRequest struct {
	CommandID   string `json:"command_id"`
	JunctionID  string `json:"junction_id"`
	Status      string `json:"status"` // "ACK" or "NACK"
	ActualState string `json:"actual_state,omitempty"`
}

// VehicleArrivalRequest represents payload when a vehicle trips a sensor.
type VehicleArrivalRequest struct {
	VehicleID   string             `json:"vehicle_id"`
	VehicleType domain.VehicleType `json:"vehicle_type"`
	Direction   domain.Direction   `json:"direction"`
}

// VehicleClearedRequest represents payload when a vehicle exits the junction.
type VehicleClearedRequest struct {
	VehicleID string           `json:"vehicle_id"`
	Direction domain.Direction `json:"direction"`
}

// EmergencyRequest represents priority vehicle preemption notification.
type EmergencyRequest struct {
	VehicleID string           `json:"vehicle_id"`
	Direction domain.Direction `json:"direction"`
}

// ManualHoldRequest represents an operator manual phase hold instruction.
type ManualHoldRequest struct {
	Phase           domain.PhaseID `json:"phase"`
	DurationSeconds int            `json:"duration_seconds"`
	User            string         `json:"user"`
}

// SimFaultRequest is used to inject faults into the simulated controller.
type SimFaultRequest struct {
	SimulateNACK *bool `json:"simulate_nack,omitempty"`
	DropACK      *bool `json:"drop_ack,omitempty"`
	Healthy      *bool `json:"healthy,omitempty"`
	LatencyMs    *int  `json:"latency_ms,omitempty"`
}
