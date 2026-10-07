package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "modernc.org/sqlite"

	"factory-traffic/internal/domain"
	"factory-traffic/internal/ports"
)

type SQLiteRepo struct {
	db *sql.DB
}

var _ ports.Repository = (*SQLiteRepo)(nil)

// NewSQLiteRepo opens a sqlite database file or in-memory instance and runs migrations.
func NewSQLiteRepo(dsn string) (*SQLiteRepo, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite database: %w", err)
	}

	// Single connection pool configuration for SQLite concurrency safety
	db.SetMaxOpenConns(1)

	repo := &SQLiteRepo{db: db}
	if err := repo.initSchema(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return repo, nil
}

func (r *SQLiteRepo) Close() error {
	return r.db.Close()
}

func (r *SQLiteRepo) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS junction_states (
		junction_id TEXT PRIMARY KEY,
		version INTEGER NOT NULL,
		mode TEXT NOT NULL,
		current_phase TEXT NOT NULL,
		current_signal TEXT NOT NULL,
		state_json TEXT NOT NULL,
		updated_at TIMESTAMP NOT NULL
	);

	CREATE TABLE IF NOT EXISTS audit_logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		junction_id TEXT NOT NULL,
		event_type TEXT NOT NULL,
		phase TEXT NOT NULL,
		mode TEXT NOT NULL,
		details_json TEXT,
		created_at TIMESTAMP NOT NULL
	);

	CREATE INDEX IF NOT EXISTS idx_audit_junction_created 
	ON audit_logs(junction_id, created_at DESC);
	`
	_, err := r.db.Exec(schema)
	return err
}

func (r *SQLiteRepo) SaveState(ctx context.Context, state domain.JunctionState) error {
	payload, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("failed to serialize state: %w", err)
	}

	query := `
	INSERT INTO junction_states (junction_id, version, mode, current_phase, current_signal, state_json, updated_at)
	VALUES (?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(junction_id) DO UPDATE SET
		version = excluded.version,
		mode = excluded.mode,
		current_phase = excluded.current_phase,
		current_signal = excluded.current_signal,
		state_json = excluded.state_json,
		updated_at = excluded.updated_at;
	`
	_, err = r.db.ExecContext(ctx, query,
		state.JunctionID,
		state.Version,
		string(state.Mode),
		string(state.CurrentPhase),
		string(state.CurrentSignal),
		string(payload),
		state.LastUpdated,
	)
	return err
}

func (r *SQLiteRepo) LoadState(ctx context.Context, junctionID string) (*domain.JunctionState, error) {
	query := `SELECT state_json FROM junction_states WHERE junction_id = ?`
	row := r.db.QueryRowContext(ctx, query, junctionID)

	var payload string
	err := row.Scan(&payload)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var state domain.JunctionState
	if err := json.Unmarshal([]byte(payload), &state); err != nil {
		return nil, fmt.Errorf("failed to parse stored state: %w", err)
	}
	return &state, nil
}

func (r *SQLiteRepo) AppendAuditLog(ctx context.Context, event domain.AuditEvent) error {
	var detailsStr string
	if event.Details != nil {
		data, err := json.Marshal(event.Details)
		if err == nil {
			detailsStr = string(data)
		}
	}

	query := `
	INSERT INTO audit_logs (junction_id, event_type, phase, mode, details_json, created_at)
	VALUES (?, ?, ?, ?, ?, ?)
	`
	_, err := r.db.ExecContext(ctx, query,
		event.JunctionID,
		event.EventType,
		string(event.Phase),
		string(event.Mode),
		detailsStr,
		event.Timestamp,
	)
	return err
}

func (r *SQLiteRepo) GetRecentAuditLogs(ctx context.Context, junctionID string, limit int) ([]domain.AuditEvent, error) {
	if limit <= 0 {
		limit = 50
	}

	query := `
	SELECT junction_id, event_type, phase, mode, details_json, created_at
	FROM audit_logs
	WHERE junction_id = ?
	ORDER BY created_at DESC
	LIMIT ?
	`
	rows, err := r.db.QueryContext(ctx, query, junctionID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []domain.AuditEvent
	for rows.Next() {
		var ev domain.AuditEvent
		var phaseStr, modeStr, detailsStr string
		var t time.Time

		if err := rows.Scan(&ev.JunctionID, &ev.EventType, &phaseStr, &modeStr, &detailsStr, &t); err != nil {
			return nil, err
		}
		ev.Phase = domain.PhaseID(phaseStr)
		ev.Mode = domain.Mode(modeStr)
		ev.Timestamp = t

		if detailsStr != "" {
			var details map[string]interface{}
			if err := json.Unmarshal([]byte(detailsStr), &details); err == nil {
				ev.Details = details
			}
		}
		logs = append(logs, ev)
	}

	return logs, rows.Err()
}
