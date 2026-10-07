package app

import (
	"context"
	"log"

	"factory-traffic/internal/domain"
	"factory-traffic/internal/ports"
)

// RecoverJunction restores junction state on boot or starts a clean safe state.
func RecoverJunction(
	ctx context.Context,
	junctionID string,
	repo ports.Repository,
	cfg domain.Config,
	clock ports.Clock,
) (domain.JunctionState, error) {
	now := clock.Now()

	if repo != nil {
		saved, err := repo.LoadState(ctx, junctionID)
		if err == nil && saved != nil {
			log.Printf("[Recovery] Found previous state for junction %s (version %d, mode %s)", junctionID, saved.Version, saved.Mode)

			recovered := saved.Clone()
			recovered.LastUpdated = now

			// If system crashed mid-transition, recover safely to All-Red clearance before resuming
			if recovered.InTransition {
				log.Printf("[Recovery] Junction %s was mid-transition during shutdown/restart; resetting to All-Red clear", junctionID)
				recovered.InTransition = false
				recovered.TransitionStep = domain.StepNone
				recovered.CurrentPhase = domain.PhaseNorthSouth
				recovered.CurrentSignal = domain.SignalGreen
				recovered.Signals = domain.SafeSignalsForPhase(domain.PhaseNorthSouth, domain.SignalGreen, cfg)
				recovered.PhaseStartedAt = now
			}

			// Clear un-acked pending command from previous process run
			recovered.PendingCommand = nil

			if err := domain.AssertSafe(recovered); err != nil {
				log.Printf("[Recovery] Saved state failed safety check (%v); initiating safe default", err)
				return domain.NewJunctionState(junctionID, cfg, now), nil
			}

			_ = repo.SaveState(ctx, recovered)
			_ = repo.AppendAuditLog(ctx, domain.AuditEvent{
				JunctionID: junctionID,
				EventType:  "SYSTEM_BOOT_RECOVERY",
				Phase:      recovered.CurrentPhase,
				Mode:       recovered.Mode,
				Details: map[string]interface{}{
					"recovered_version": recovered.Version,
				},
				Timestamp: now,
			})

			return recovered, nil
		}
	}

	// No prior state: initialize fresh safe default
	fresh := domain.NewJunctionState(junctionID, cfg, now)
	if repo != nil {
		_ = repo.SaveState(ctx, fresh)
		_ = repo.AppendAuditLog(ctx, domain.AuditEvent{
			JunctionID: junctionID,
			EventType:  "SYSTEM_INITIAL_BOOT",
			Phase:      fresh.CurrentPhase,
			Mode:       fresh.Mode,
			Details: map[string]interface{}{
				"initial_phase": fresh.CurrentPhase,
			},
			Timestamp: now,
		})
	}

	return fresh, nil
}
