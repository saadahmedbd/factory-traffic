package domain

import (
	"testing"
	"time"
)

func TestControllerCommandFields(t *testing.T) {
	now := time.Now()
	cmd := ControllerCommand{
		ID:         "cmd-1",
		JunctionID: "j1",
		Seq:        10,
		Type:       CommandSetSignals,
		Signals: map[Direction]Signal{
			DirectionNorth: SignalGreen,
			DirectionSouth: SignalGreen,
			DirectionEast:  SignalRed,
			DirectionWest:  SignalRed,
		},
		IssuedAt: now,
		Deadline: now.Add(2 * time.Second),
		Status:   CommandStatusPending,
	}

	if cmd.Status != CommandStatusPending {
		t.Errorf("expected pending status, got %s", cmd.Status)
	}

	ack := ControllerAck{
		CommandID:  "cmd-1",
		JunctionID: "j1",
		Seq:        10,
		Success:    true,
		Timestamp:  now.Add(100 * time.Millisecond),
	}

	if !ack.Success || ack.CommandID != cmd.ID {
		t.Errorf("ack mismatch")
	}
}
