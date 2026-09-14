-- +goose Up
CREATE TABLE IF NOT EXISTS backup_runs (
    id TEXT PRIMARY KEY,
    database_name TEXT NOT NULL,
    engine TEXT NOT NULL,
    status TEXT NOT NULL,
    started_at DATETIME NOT NULL,
    completed_at DATETIME,
    duration_ms INTEGER DEFAULT 0,
    size_bytes INTEGER DEFAULT 0,
    destinations TEXT DEFAULT '[]',
    error_message TEXT DEFAULT '',
    logs TEXT DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_backup_runs_db ON backup_runs(database_name);
CREATE INDEX IF NOT EXISTS idx_backup_runs_started ON backup_runs(started_at DESC);
CREATE INDEX IF NOT EXISTS idx_backup_runs_status ON backup_runs(status);

-- +goose Down
DROP TABLE IF EXISTS backup_runs;
