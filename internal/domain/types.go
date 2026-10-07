package domain

import (
	"fmt"
	"time"
)

// Direction represents a physical traffic approach direction.
type Direction string

const (
	DirectionNorth Direction = "NORTH"
	DirectionSouth Direction = "SOUTH"
	DirectionEast  Direction = "EAST"
	DirectionWest  Direction = "WEST"
)

// AllDirections lists all supported junction approaches.
var AllDirections = []Direction{
	DirectionNorth,
	DirectionSouth,
	DirectionEast,
	DirectionWest,
}

// Signal represents the operational color/state of a traffic light.
type Signal string

const (
	SignalRed    Signal = "RED"
	SignalYellow Signal = "YELLOW"
	SignalGreen  Signal = "GREEN"
)

// Mode defines the system operating mode.
type Mode string

const (
	ModeAuto      Mode = "AUTO"
	ModeManual    Mode = "MANUAL"
	ModeEmergency Mode = "EMERGENCY"
	ModeDegraded  Mode = "DEGRADED" // Failsafe / Yellow flash mode
)

// VehicleType classifies vehicles inside the factory floor.
type VehicleType string

const (
	VehicleNormal          VehicleType = "NORMAL"
	VehicleForklift        VehicleType = "FORKLIFT"
	VehicleTruck           VehicleType = "TRUCK"
	VehicleEmployee        VehicleType = "EMPLOYEE_VEHICLE"
	VehicleAGV             VehicleType = "AGV"
	VehicleEmergency       VehicleType = "EMERGENCY"
)

// PhaseID identifies an active traffic phase.
type PhaseID string

const (
	PhaseNorthSouth PhaseID = "PHASE_NORTH_SOUTH"
	PhaseEastWest   PhaseID = "PHASE_EAST_WEST"
	PhaseAllRed     PhaseID = "PHASE_ALL_RED"
)

// Phase defines an allowable set of non-conflicting directions.
type Phase struct {
	ID                PhaseID     `json:"id"`
	Name              string      `json:"name"`
	AllowedDirections []Direction `json:"allowed_directions"`
}

// Vehicle represents an individual vehicle detected in a queue.
type Vehicle struct {
	ID        string      `json:"id"`
	Type      VehicleType `json:"type"`
	Direction Direction   `json:"direction"`
	ArrivedAt time.Time   `json:"arrived_at"`
	Weight    int         `json:"weight"`
}

// TransitionStep represents sub-states during phase switching.
type TransitionStep string

const (
	StepNone          TransitionStep = "NONE"
	StepYellowClear   TransitionStep = "YELLOW_CLEAR"
	StepAllRedClear   TransitionStep = "ALL_RED_CLEAR"
)

// IsValidDirection checks if a direction string is known.
func IsValidDirection(d Direction) bool {
	switch d {
	case DirectionNorth, DirectionSouth, DirectionEast, DirectionWest:
		return true
	default:
		return false
	}
}

// ParseDirection parses a string into a valid Direction or returns error.
func ParseDirection(s string) (Direction, error) {
	d := Direction(s)
	if IsValidDirection(d) {
		return d, nil
	}
	return "", fmt.Errorf("invalid direction: %s", s)
}
