package simcontroller

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"factory-traffic/internal/domain"
	"factory-traffic/internal/ports"
)

// AckCallbackFunc is a callback invoked when the simulated hardware sends an ACK.
type AckCallbackFunc func(ack domain.ControllerAck)

// SimController simulates a hardware traffic controller with realistic actuation delay and fault injection.
type SimController struct {
	mu           sync.RWMutex
	signals      map[domain.Direction]domain.Signal
	lastCommand  *domain.ControllerCommand
	ackCallback  AckCallbackFunc
	healthy      bool
	simulateNACK bool
	dropACK      bool
	latency      time.Duration
}

var _ ports.ControllerPort = (*SimController)(nil)

// NewSimController creates a simulated controller.
func NewSimController(ackCallback AckCallbackFunc) *SimController {
	signals := make(map[domain.Direction]domain.Signal)
	for _, d := range domain.AllDirections {
		signals[d] = domain.SignalRed
	}
	// Initial default lights: NS Green, EW Red
	signals[domain.DirectionNorth] = domain.SignalGreen
	signals[domain.DirectionSouth] = domain.SignalGreen

	return &SimController{
		signals:      signals,
		ackCallback:  ackCallback,
		healthy:      true,
		simulateNACK: false,
		dropACK:      false,
		latency:      50 * time.Millisecond,
	}
}

// SendCommand accepts instructions, applies hardware state, and dispatches an asynchronous ACK.
func (s *SimController) SendCommand(ctx context.Context, cmd domain.ControllerCommand) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.healthy {
		return fmt.Errorf("controller hardware offline")
	}

	cmdCopy := cmd
	s.lastCommand = &cmdCopy

	// Apply physical signals
	for dir, sig := range cmd.Signals {
		s.signals[dir] = sig
	}

	log.Printf("[SimController] Applied Command: %s (Type: %s) -> North:%s, East:%s",
		cmd.ID, cmd.Type, s.signals[domain.DirectionNorth], s.signals[domain.DirectionEast])

	// Dispatch ACK asynchronously
	if s.ackCallback != nil && !s.dropACK {
		go func(c domain.ControllerCommand, shouldFail bool, delay time.Duration) {
			if delay > 0 {
				time.Sleep(delay)
			}
			ack := domain.ControllerAck{
				CommandID:  c.ID,
				JunctionID: c.JunctionID,
				Seq:        c.Seq,
				Success:    !shouldFail,
				Timestamp:  time.Now(),
			}
			if shouldFail {
				ack.ErrorMessage = "Simulated hardware coil failure"
			}
			s.ackCallback(ack)
		}(cmd, s.simulateNACK, s.latency)
	}

	return nil
}

func (s *SimController) HealthCheck(ctx context.Context) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if !s.healthy {
		return fmt.Errorf("controller hardware reporting error")
	}
	return nil
}

// GetSignals returns the current physical signal output.
func (s *SimController) GetSignals() map[domain.Direction]domain.Signal {
	s.mu.RLock()
	defer s.mu.RUnlock()
	res := make(map[domain.Direction]domain.Signal)
	for k, v := range s.signals {
		res[k] = v
	}
	return res
}

// SetSimulateNACK enables or disables hardware rejection.
func (s *SimController) SetSimulateNACK(fail bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.simulateNACK = fail
}

// SetDropACK simulates communication drop / timeout.
func (s *SimController) SetDropACK(drop bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.dropACK = drop
}

// SetHealthy toggles controller health.
func (s *SimController) SetHealthy(healthy bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.healthy = healthy
}

// SetLatency sets simulated execution latency.
func (s *SimController) SetLatency(d time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.latency = d
}
