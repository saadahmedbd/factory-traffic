package app

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"factory-traffic/internal/domain"
	"factory-traffic/internal/ports"
)

type mockController struct{}

func (m *mockController) SendCommand(ctx context.Context, cmd domain.ControllerCommand) error {
	return nil
}

func (m *mockController) HealthCheck(ctx context.Context) error {
	return nil
}

type mockRepo struct {
	mu     sync.Mutex
	states map[string]domain.JunctionState
	logs   []domain.AuditEvent
}

func newMockRepo() *mockRepo {
	return &mockRepo{
		states: make(map[string]domain.JunctionState),
		logs:   make([]domain.AuditEvent, 0),
	}
}

func (m *mockRepo) SaveState(ctx context.Context, state domain.JunctionState) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.states[state.JunctionID] = state
	return nil
}

func (m *mockRepo) LoadState(ctx context.Context, junctionID string) (*domain.JunctionState, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if s, ok := m.states[junctionID]; ok {
		cpy := s.Clone()
		return &cpy, nil
	}
	return nil, nil
}

func (m *mockRepo) AppendAuditLog(ctx context.Context, event domain.AuditEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logs = append(m.logs, event)
	return nil
}

func (m *mockRepo) GetRecentAuditLogs(ctx context.Context, junctionID string, limit int) ([]domain.AuditEvent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.logs, nil
}

func TestActorConcurrencyScenario(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	clock := ports.NewRealClock()
	ctrl := &mockController{}
	repo := newMockRepo()
	cfg := domain.DefaultConfig()

	initialState := domain.NewJunctionState("junction-concurrent", cfg, clock.Now())
	actor := NewJunctionActor(initialState, ctrl, repo, clock)
	actor.Start(ctx)
	defer actor.Stop()

	// Concurrently send 50 arrivals from various goroutines
	var wg sync.WaitGroup
	concurrentCount := 50

	for i := 0; i < concurrentCount; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			dir := domain.DirectionNorth
			if idx%2 == 0 {
				dir = domain.DirectionEast
			}
			v := domain.Vehicle{
				ID:        fmt.Sprintf("AGV-%d", idx),
				Type:      domain.VehicleAGV,
				Direction: dir,
				ArrivedAt: clock.Now(),
			}

			err := actor.Send(ctx, domain.EngineInput{
				Type:    domain.InputTypeVehicleArrival,
				Vehicle: &v,
			})
			if err != nil {
				t.Errorf("actor send failed: %v", err)
			}

			// Concurrently read state
			st := actor.GetState()
			if st.JunctionID != "junction-concurrent" {
				t.Errorf("corrupt state observed")
			}
		}(i)
	}

	wg.Wait()

	finalState := actor.GetState()
	if finalState.JunctionID != "junction-concurrent" {
		t.Errorf("expected junction-concurrent")
	}
}

func TestBootTimeRecoveryScenario(t *testing.T) {
	ctx := context.Background()
	clock := ports.NewRealClock()
	repo := newMockRepo()
	cfg := domain.DefaultConfig()

	// 1. Initial boot (no repo state yet)
	s1, err := RecoverJunction(ctx, "j-recov", repo, cfg, clock)
	if err != nil {
		t.Fatalf("recovery failed: %v", err)
	}
	if s1.CurrentPhase != domain.PhaseNorthSouth {
		t.Errorf("expected default PhaseNorthSouth, got %s", s1.CurrentPhase)
	}

	// 2. Simulate dirty shutdown mid-transition
	s1.InTransition = true
	s1.TransitionStep = domain.StepYellowClear
	_ = repo.SaveState(ctx, s1)

	// 3. Reboot: recovery should reset mid-transition to safe steady state
	s2, err := RecoverJunction(ctx, "j-recov", repo, cfg, clock)
	if err != nil {
		t.Fatalf("recovery failed: %v", err)
	}
	if s2.InTransition {
		t.Errorf("expected recovered state to clear InTransition flag")
	}
	if err := domain.AssertSafe(s2); err != nil {
		t.Errorf("recovered state violates safety invariant: %v", err)
	}
}
