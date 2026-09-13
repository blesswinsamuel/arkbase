package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

type Store struct {
	db *sql.DB
}

type Run struct {
	ID           string     `json:"id"`
	DatabaseName string     `json:"database_name"`
	Engine       string     `json:"engine"`
	Status       string     `json:"status"` // "running", "success", "failed"
	StartedAt    time.Time  `json:"started_at"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
	DurationMs   int64      `json:"duration_ms"`
	SizeBytes    int64      `json:"size_bytes"`
	Destinations []string   `json:"destinations"`
	ErrorMessage string     `json:"error_message,omitempty"`
	Logs         string     `json:"logs,omitempty"`
}

type SummaryStats struct {
	TotalDatabases  int   `json:"total_databases"`
	TotalBackups    int   `json:"total_backups"`
	SuccessfulCount int   `json:"successful_count"`
	FailedCount     int   `json:"failed_count"`
	TotalSizeBytes  int64 `json:"total_size_bytes"`
	LastBackupAt    *time.Time `json:"last_backup_at,omitempty"`
}

func Open(dataDir string) (*Store, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}

	dbPath := filepath.Join(dataDir, "arkbase.db")
	db, err := sql.Open("sqlite", dbPath+"?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)")
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	db.SetMaxOpenConns(1) // SQLite is best with 1 writer

	schema := `
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
	`
	if _, err := db.Exec(schema); err != nil {
		return nil, fmt.Errorf("migrate sqlite schema: %w", err)
	}

	return &Store{db: db}, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) RecordStart(ctx context.Context, id, dbName, engine string) error {
	query := `INSERT INTO backup_runs (id, database_name, engine, status, started_at) VALUES (?, ?, ?, 'running', ?)`
	_, err := s.db.ExecContext(ctx, query, id, dbName, engine, time.Now().UTC())
	return err
}

func (s *Store) RecordFinish(ctx context.Context, id, status string, sizeBytes int64, duration time.Duration, destinations []string, errorMsg, logs string) error {
	destsJSON, _ := json.Marshal(destinations)
	now := time.Now().UTC()
	query := `
	UPDATE backup_runs 
	SET status = ?, completed_at = ?, duration_ms = ?, size_bytes = ?, destinations = ?, error_message = ?, logs = ?
	WHERE id = ?`
	_, err := s.db.ExecContext(ctx, query, status, now, duration.Milliseconds(), sizeBytes, string(destsJSON), errorMsg, logs, id)
	return err
}

func (s *Store) GetRecentRuns(ctx context.Context, limit, offset int, dbFilter string) ([]Run, int, error) {
	if limit <= 0 {
		limit = 20
	}

	var countQuery string
	var query string
	var args []any
	var countArgs []any

	if dbFilter != "" {
		countQuery = `SELECT COUNT(*) FROM backup_runs WHERE database_name = ?`
		query = `SELECT id, database_name, engine, status, started_at, completed_at, duration_ms, size_bytes, destinations, error_message, logs FROM backup_runs WHERE database_name = ? ORDER BY started_at DESC LIMIT ? OFFSET ?`
		countArgs = append(countArgs, dbFilter)
		args = append(args, dbFilter, limit, offset)
	} else {
		countQuery = `SELECT COUNT(*) FROM backup_runs`
		query = `SELECT id, database_name, engine, status, started_at, completed_at, duration_ms, size_bytes, destinations, error_message, logs FROM backup_runs ORDER BY started_at DESC LIMIT ? OFFSET ?`
		args = append(args, limit, offset)
	}

	var total int
	if err := s.db.QueryRowContext(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var runs []Run
	for rows.Next() {
		var r Run
		var destsStr string
		var completedAt sql.NullTime

		if err := rows.Scan(&r.ID, &r.DatabaseName, &r.Engine, &r.Status, &r.StartedAt, &completedAt, &r.DurationMs, &r.SizeBytes, &destsStr, &r.ErrorMessage, &r.Logs); err != nil {
			return nil, 0, err
		}
		if completedAt.Valid {
			r.CompletedAt = &completedAt.Time
		}
		_ = json.Unmarshal([]byte(destsStr), &r.Destinations)
		runs = append(runs, r)
	}

	return runs, total, nil
}

func (s *Store) GetRun(ctx context.Context, id string) (*Run, error) {
	query := `SELECT id, database_name, engine, status, started_at, completed_at, duration_ms, size_bytes, destinations, error_message, logs FROM backup_runs WHERE id = ?`
	var r Run
	var destsStr string
	var completedAt sql.NullTime

	err := s.db.QueryRowContext(ctx, query, id).Scan(&r.ID, &r.DatabaseName, &r.Engine, &r.Status, &r.StartedAt, &completedAt, &r.DurationMs, &r.SizeBytes, &destsStr, &r.ErrorMessage, &r.Logs)
	if err != nil {
		return nil, err
	}
	if completedAt.Valid {
		r.CompletedAt = &completedAt.Time
	}
	_ = json.Unmarshal([]byte(destsStr), &r.Destinations)
	return &r, nil
}

func (s *Store) GetSummaryStats(ctx context.Context) (*SummaryStats, error) {
	stats := &SummaryStats{}

	row := s.db.QueryRowContext(ctx, `
		SELECT 
			COUNT(*),
			COALESCE(SUM(CASE WHEN status = 'success' THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN status = 'success' THEN size_bytes ELSE 0 END), 0),
			MAX(started_at)
		FROM backup_runs
	`)
	var lastBackup sql.NullString
	if err := row.Scan(&stats.TotalBackups, &stats.SuccessfulCount, &stats.FailedCount, &stats.TotalSizeBytes, &lastBackup); err != nil {
		return nil, err
	}
	if lastBackup.Valid && lastBackup.String != "" {
		for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02 15:04:05.999999999-07:00", "2006-01-02 15:04:05-07:00", "2006-01-02 15:04:05"} {
			if t, err := time.Parse(layout, lastBackup.String); err == nil {
				stats.LastBackupAt = &t
				break
			}
		}
	}

	rowDBs := s.db.QueryRowContext(ctx, `SELECT COUNT(DISTINCT database_name) FROM backup_runs`)
	_ = rowDBs.Scan(&stats.TotalDatabases)

	return stats, nil
}
