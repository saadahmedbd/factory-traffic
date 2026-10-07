package domain

import "time"

// Config encapsulates timings, weights, and phase safety configuration.
type Config struct {
	MinGreenDuration       time.Duration          `json:"min_green_duration"`
	MaxGreenDuration       time.Duration          `json:"max_green_duration"`
	YellowDuration         time.Duration          `json:"yellow_duration"`
	AllRedDuration         time.Duration          `json:"all_red_duration"`
	StarvationThreshold    time.Duration          `json:"starvation_threshold"`
	StarvationBonus        int                    `json:"starvation_bonus"`
	ManualHoldMaxDuration  time.Duration          `json:"manual_hold_max_duration"`
	EmergencyTTL           time.Duration          `json:"emergency_ttl"`
	DeduplicationWindow    time.Duration          `json:"deduplication_window"`
	AckTimeout             time.Duration          `json:"ack_timeout"`
	MaxAckRetries          int                    `json:"max_ack_retries"`
	VehicleWeights         map[VehicleType]int    `json:"vehicle_weights"`
	Phases                 map[PhaseID]Phase      `json:"phases"`
}

// DefaultConfig returns safe, industry-tested traffic parameters for industrial floor automation.
func DefaultConfig() Config {
	return Config{
		MinGreenDuration:      5 * time.Second,
		MaxGreenDuration:      30 * time.Second,
		YellowDuration:        5 * time.Second,
		AllRedDuration:        2 * time.Second,
		StarvationThreshold:   15 * time.Second,
		StarvationBonus:       10,
		ManualHoldMaxDuration: 60 * time.Second,
		EmergencyTTL:          45 * time.Second,
		DeduplicationWindow:   3 * time.Second,
		AckTimeout:            2 * time.Second,
		MaxAckRetries:         3,
		VehicleWeights: map[VehicleType]int{
			VehicleEmergency: 100,
			VehicleTruck:     5,
			VehicleAGV:       5,
			VehicleForklift:  3,
			VehicleEmployee:  1,
			VehicleNormal:    1,
		},
		Phases: map[PhaseID]Phase{
			PhaseNorthSouth: {
				ID:   PhaseNorthSouth,
				Name: "North-South Corridor",
				AllowedDirections: []Direction{
					DirectionNorth,
					DirectionSouth,
				},
			},
			PhaseEastWest: {
				ID:   PhaseEastWest,
				Name: "East-West Corridor",
				AllowedDirections: []Direction{
					DirectionEast,
					DirectionWest,
				},
			},
			PhaseAllRed: {
				ID:                PhaseAllRed,
				Name:              "All-Red Clearance",
				AllowedDirections: []Direction{},
			},
		},
	}
}

// WeightFor returns the weight assigned to a specific vehicle type.
func (c Config) WeightFor(vt VehicleType) int {
	if w, ok := c.VehicleWeights[vt]; ok {
		return w
	}
	return 1
}
