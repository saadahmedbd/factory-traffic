package sqlite

import (
	"context"
	"testing"
	"time"

	"factory-traffic/internal/domain"
)

func TestSQLiteRepoSaveAndLoad(t *testing.T) {
	ctx := context.Background()
	// Use pure in-memory SQLite database
	repo, err := NewSQLiteRepo(":memory:")
	if err != nil {
		t.Fatalf("failed to create in-memory sqlite repo: %v", err)
	}
	defer repo.Close()

	now := time.Now().Truncate(time.Millisecond)
	cfg := domain.DefaultConfig()
	state := domain.NewJunctionState("junction-sql", cfg, now)

	// Save
	if err := repo.SaveState(ctx, state); err != nil {
		t.Fatalf("failed to save state: %v", err)
	}

	// Load
	loaded, err := repo.LoadState(ctx, "junction-sql")
	if err != nil {
		t.Fatalf("failed to load state: %v", err)
	}
	if loaded == nil {
		t.Fatalf("expected loaded state, got nil")
	}
	if loaded.JunctionID != "junction-sql" {
		t.Errorf("expected junction-sql, got %s", loaded.JunctionID)
	}
	if loaded.CurrentPhase != domain.PhaseNorthSouth {
		t.Errorf("expected PhaseNorthSouth, got %s", loaded.CurrentPhase)
	}

	// Test non-existent junction
	missing, err := repo.LoadState(ctx, "non-existent")
	if err != nil || missing != nil {
		t.Errorf("expected nil for missing junction, got %v, err=%v", missing, err)
	}
}

func TestSQLiteRepoAuditLogs(t *testing.T) {
	ctx := context.Background()
	repo, err := NewSQLiteRepo(":memory:")
	if err != nil {
		t.Fatalf("failed to initialize repo: %v", err)
	}
	defer repo.Close()

	now := time.Now().Truncate(time.Millisecond)
	ev1 := domain.AuditEvent{
		JunctionID: "j-test",
		EventType:  "ARRIVAL",
		Phase:      domain.PhaseNorthSouth,
		Mode:       domain.ModeAuto,
		Details:    map[string]interface{}{"vehicle": "AGV-1"},
		Timestamp:  now,
	}
	ev2 := domain.AuditEvent{
		JunctionID: "j-test",
		EventType:  "DEPARTURE",
		Phase:      domain.PhaseNorthSouth,
		Mode:       domain.ModeAuto,
		Details:    map[string]interface{}{"vehicle": "AGV-1"},
		Timestamp:  now.Add(5 * time.Second),
	}

	if err := repo.AppendAuditLog(ctx, ev1); err != nil {
		t.Fatalf("failed to append ev1: %v", err)
	}
	if err := repo.AppendAuditLog(ctx, ev2); err != nil {
		t.Fatalf("failed to append ev2: %v", err)
	}

	logs, err := repo.GetRecentAuditLogs(ctx, "j-test", 10)
	if err != nil {
		t.Fatalf("failed to query audit logs: %v", err)
	}
	if len(logs) != 2 {
		t.Fatalf("expected 2 logs, got %d", len(logs))
	}
	// ev2 was more recent, should be first in DESC order
	if logs[0].EventType != "DEPARTURE" {
		t.Errorf("expected DEPARTURE first, got %s", logs[0].EventType)
	}
}
