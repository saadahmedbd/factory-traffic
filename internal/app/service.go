package app

import (
	"context"
	"fmt"
	"sync"
	"time"

	"factory-traffic/internal/domain"
	"factory-traffic/internal/ports"
)

// TrafficService orchestrates the junction actors and coordinates use case operations.
type TrafficService struct {
	mu      sync.RWMutex
	actors  map[string]*JunctionActor
	ctrl    ports.ControllerPort
	repo    ports.Repository
	clock   ports.Clock
}

// NewTrafficService creates a new TrafficService instance.
func NewTrafficService(ctrl ports.ControllerPort, repo ports.Repository, clock ports.Clock) *TrafficService {
	return &TrafficService{
		actors: make(map[string]*JunctionActor),
		ctrl:   ctrl,
		repo:   repo,
		clock:  clock,
	}
}

// RegisterJunction creates or registers an actor for a junction.
func (s *TrafficService) RegisterJunction(ctx context.Context, state domain.JunctionState) *JunctionActor {
	s.mu.Lock()
	defer s.mu.Unlock()

	if actor, exists := s.actors[state.JunctionID]; exists {
		return actor
	}

	actor := NewJunctionActor(state, s.ctrl, s.repo, s.clock)
	actor.Start(ctx)
	s.actors[state.JunctionID] = actor
	return actor
}

// GetActor looks up an active junction actor.
func (s *TrafficService) GetActor(junctionID string) (*JunctionActor, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	actor, exists := s.actors[junctionID]
	if !exists {
		return nil, fmt.Errorf("junction '%s' not found", junctionID)
	}
	return actor, nil
}

// RecordArrival handles sensor/RFID vehicle detection.
func (s *TrafficService) RecordArrival(ctx context.Context, junctionID string, v domain.Vehicle) error {
	actor, err := s.GetActor(junctionID)
	if err != nil {
		return err
	}
	if v.ArrivedAt.IsZero() {
		v.ArrivedAt = s.clock.Now()
	}
	return actor.Send(ctx, domain.EngineInput{
		Type:    domain.InputTypeVehicleArrival,
		Vehicle: &v,
	})
}

// RecordCleared handles vehicle clearance after traversing the junction.
func (s *TrafficService) RecordCleared(ctx context.Context, junctionID string, dir domain.Direction, vehicleID string) error {
	actor, err := s.GetActor(junctionID)
	if err != nil {
		return err
	}
	return actor.Send(ctx, domain.EngineInput{
		Type:             domain.InputTypeVehicleCleared,
		ClearedVehicleID: vehicleID,
		ClearedDirection: dir,
	})
}

// TriggerEmergency handles priority preemption for emergency vehicles.
func (s *TrafficService) TriggerEmergency(ctx context.Context, junctionID string, v domain.Vehicle) error {
	actor, err := s.GetActor(junctionID)
	if err != nil {
		return err
	}
	if v.ArrivedAt.IsZero() {
		v.ArrivedAt = s.clock.Now()
	}
	return actor.Send(ctx, domain.EngineInput{
		Type:    domain.InputTypeEmergencyPreempt,
		Vehicle: &v,
	})
}

// ClearEmergency clears an emergency vehicle once it has passed.
func (s *TrafficService) ClearEmergency(ctx context.Context, junctionID string, vehicleID string) error {
	actor, err := s.GetActor(junctionID)
	if err != nil {
		return err
	}
	return actor.Send(ctx, domain.EngineInput{
		Type:             domain.InputTypeEmergencyCleared,
		ClearedVehicleID: vehicleID,
	})
}

// SetManualOverride activates manual phase hold for an operator.
func (s *TrafficService) SetManualOverride(ctx context.Context, junctionID string, phase domain.PhaseID, duration time.Duration, user string) error {
	actor, err := s.GetActor(junctionID)
	if err != nil {
		return err
	}
	return actor.Send(ctx, domain.EngineInput{
		Type:           domain.InputTypeManualOverride,
		ManualPhase:    phase,
		ManualDuration: duration,
		ManualUser:     user,
	})
}

// ReleaseManualOverride releases an active manual hold.
func (s *TrafficService) ReleaseManualOverride(ctx context.Context, junctionID string) error {
	actor, err := s.GetActor(junctionID)
	if err != nil {
		return err
	}
	return actor.Send(ctx, domain.EngineInput{
		Type: domain.InputTypeReleaseManual,
	})
}

// ProcessControllerAck processes hardware controller feedback.
func (s *TrafficService) ProcessControllerAck(ctx context.Context, ack domain.ControllerAck) error {
	actor, err := s.GetActor(ack.JunctionID)
	if err != nil {
		return err
	}
	return actor.Send(ctx, domain.EngineInput{
		Type: domain.InputTypeControllerAck,
		Ack:  &ack,
	})
}

// GetJunctionStatus retrieves current junction status.
func (s *TrafficService) GetJunctionStatus(junctionID string) (domain.JunctionState, error) {
	actor, err := s.GetActor(junctionID)
	if err != nil {
		return domain.JunctionState{}, err
	}
	return actor.GetState(), nil
}

// GetAuditHistory queries historical audit records for a junction.
func (s *TrafficService) GetAuditHistory(ctx context.Context, junctionID string, limit int) ([]domain.AuditEvent, error) {
	if s.repo == nil {
		return nil, nil
	}
	return s.repo.GetRecentAuditLogs(ctx, junctionID, limit)
}

// ListJunctions returns the list of registered junction identifiers.
func (s *TrafficService) ListJunctions() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	res := make([]string, 0, len(s.actors))
	for id := range s.actors {
		res = append(res, id)
	}
	return res
}

// ProcessSensorEvent handles general sensor events (VEHICLE_ARRIVED, VEHICLE_CLEARED) with event_id idempotency.
func (s *TrafficService) ProcessSensorEvent(
	ctx context.Context,
	eventID string,
	junctionID string,
	dir domain.Direction,
	eventType string,
	vehicleID string,
	vehicleType domain.VehicleType,
	seqNo int64,
	timestamp time.Time,
) error {
	actor, err := s.GetActor(junctionID)
	if err != nil {
		return err
	}

	if timestamp.IsZero() {
		timestamp = s.clock.Now()
	}

	if eventType == "VEHICLE_CLEARED" {
		return actor.Send(ctx, domain.EngineInput{
			Type:             domain.InputTypeVehicleCleared,
			EventID:          eventID,
			SequenceNo:       seqNo,
			ClearedVehicleID: vehicleID,
			ClearedDirection: dir,
		})
	}

	// Default: VEHICLE_ARRIVED
	v := domain.Vehicle{
		ID:        vehicleID,
		Type:      vehicleType,
		Direction: dir,
		ArrivedAt: timestamp,
	}

	if vehicleType == domain.VehicleEmergency {
		return actor.Send(ctx, domain.EngineInput{
			Type:       domain.InputTypeEmergencyPreempt,
			EventID:    eventID,
			SequenceNo: seqNo,
			Vehicle:    &v,
		})
	}

	return actor.Send(ctx, domain.EngineInput{
		Type:       domain.InputTypeVehicleArrival,
		EventID:    eventID,
		SequenceNo: seqNo,
		Vehicle:    &v,
	})
}

// StopAll stops all active junction actors.
func (s *TrafficService) StopAll() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, a := range s.actors {
		a.Stop()
	}
}
