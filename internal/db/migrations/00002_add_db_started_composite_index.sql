-- +goose Up
CREATE INDEX IF NOT EXISTS idx_backup_runs_db_started ON backup_runs(database_name, started_at DESC);

-- +goose Down
DROP INDEX IF EXISTS idx_backup_runs_db_started;
