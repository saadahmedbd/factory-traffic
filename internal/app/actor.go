package app

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"factory-traffic/internal/domain"
	"factory-traffic/internal/ports"
)

// JunctionActor manages the isolated state of a single traffic junction using the Actor pattern.
type JunctionActor struct {
	junctionID string
	state      domain.JunctionState
	inbox      chan actorEnvelope
	ctrl       ports.ControllerPort
	repo       ports.Repository
	clock      ports.Clock
	dedupe     *domain.DedupeTracker
	stopCh     chan struct{}
	wg         sync.WaitGroup
	mu         sync.RWMutex
}

type actorEnvelope struct {
	input    domain.EngineInput
	respChan chan error
}

// NewJunctionActor instantiates an actor for a junction.
func NewJunctionActor(
	initialState domain.JunctionState,
	ctrl ports.ControllerPort,
	repo ports.Repository,
	clock ports.Clock,
) *JunctionActor {
	return &JunctionActor{
		junctionID: initialState.JunctionID,
		state:      initialState,
		inbox:      make(chan actorEnvelope, 256),
		ctrl:       ctrl,
		repo:       repo,
		clock:      clock,
		dedupe:     domain.NewDedupeTracker(),
		stopCh:     make(chan struct{}),
	}
}

// Start launches the actor's event loop in a background goroutine.
func (a *JunctionActor) Start(ctx context.Context) {
	a.wg.Add(1)
	go a.loop(ctx)
}

// Stop gracefully shuts down the actor.
func (a *JunctionActor) Stop() {
	close(a.stopCh)
	a.wg.Wait()
}

// Send submits an engine input and waits for it to be processed.
func (a *JunctionActor) Send(ctx context.Context, input domain.EngineInput) error {
	resp := make(chan error, 1)
	select {
	case <-a.stopCh:
		return fmt.Errorf("actor for junction %s is stopped", a.junctionID)
	case <-ctx.Done():
		return ctx.Err()
	case a.inbox <- actorEnvelope{input: input, respChan: resp}:
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-resp:
		return err
	}
}

// GetState returns a snapshot of the current state thread-safely.
func (a *JunctionActor) GetState() domain.JunctionState {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.state.Clone()
}

func (a *JunctionActor) loop(ctx context.Context) {
	defer a.wg.Done()
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-a.stopCh:
			return

		case now := <-ticker.C:
			// Regular maintenance tick
			a.processInput(ctx, domain.EngineInput{Type: domain.InputTypeTick}, now, nil)

		case env := <-a.inbox:
			now := a.clock.Now()
			a.processInput(ctx, env.input, now, env.respChan)
		}
	}
}

func (a *JunctionActor) processInput(ctx context.Context, input domain.EngineInput, now time.Time, resp chan<- error) {
	// Idempotency check: duplicate event_id must not modify state multiple times (PDF Section 4)
	if input.EventID != "" && a.dedupe.IsEventProcessed(input.EventID) {
		if resp != nil {
			resp <- nil
		}
		return
	}

	// Deduplication check for vehicle sensor bounce
	if input.Type == domain.InputTypeVehicleArrival && input.Vehicle != nil {
		if a.dedupe.IsDuplicate(input.Vehicle.ID, now, a.state.Config.DeduplicationWindow) {
			if resp != nil {
				resp <- nil
			}
			return
		}
	}

	a.mu.Lock()
	newState, effects, err := domain.Decide(a.state, input, now)
	if err != nil {
		a.mu.Unlock()
		if resp != nil {
			resp <- err
		}
		return
	}
	a.state = newState
	a.mu.Unlock()

	if resp != nil {
		resp <- nil
	}

	// Execute side-effects asynchronously or sequentially outside state lock
	for _, eff := range effects {
		a.executeEffect(ctx, eff)
	}
}

func (a *JunctionActor) executeEffect(ctx context.Context, eff domain.Effect) {
	switch eff.Type {
	case domain.EffectTypeSendCommand:
		if eff.Command != nil && a.ctrl != nil {
			if err := a.ctrl.SendCommand(ctx, *eff.Command); err != nil {
				log.Printf("[Junction %s] error dispatching hardware command %s: %v", a.junctionID, eff.Command.ID, err)
			}
		}

	case domain.EffectTypePersist:
		if a.repo != nil {
			curr := a.GetState()
			if err := a.repo.SaveState(ctx, curr); err != nil {
				log.Printf("[Junction %s] error persisting state: %v", a.junctionID, err)
			}
		}

	case domain.EffectTypeAudit:
		if eff.Audit != nil && a.repo != nil {
			if err := a.repo.AppendAuditLog(ctx, *eff.Audit); err != nil {
				log.Printf("[Junction %s] error appending audit log: %v", a.junctionID, err)
			}
		}

	case domain.EffectTypeScheduleTimer:
		// Timer events will naturally be evaluated on periodic ticks or delayed message
		if eff.TimerDuration > 0 {
			go func(d time.Duration, reason string) {
				time.Sleep(d)
				_ = a.Send(context.Background(), domain.EngineInput{Type: domain.InputTypeTick})
			}(eff.TimerDuration, eff.TimerReason)
		}
	}
}
