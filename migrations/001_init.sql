-- Migration 001: Initial schema for factory traffic controller

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
