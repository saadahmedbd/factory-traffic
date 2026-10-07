package ports

import (
	"context"
	"factory-traffic/internal/domain"
)

// Repository defines data persistence capabilities for state snapshots and audit history.
type Repository interface {
	// SaveState persists the state of a junction.
	SaveState(ctx context.Context, state domain.JunctionState) error

	// LoadState retrieves the most recent saved state for a junction.
	LoadState(ctx context.Context, junctionID string) (*domain.JunctionState, error)

	// AppendAuditLog appends an audit event to the persistent journal.
	AppendAuditLog(ctx context.Context, event domain.AuditEvent) error

	// GetRecentAuditLogs returns recent audit events for a junction up to limit.
	GetRecentAuditLogs(ctx context.Context, junctionID string, limit int) ([]domain.AuditEvent, error)
}
