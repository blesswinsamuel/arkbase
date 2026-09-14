package db_test

import (
	"context"
	"testing"
	"time"

	"github.com/blesswinsamuel/arkbase/internal/db"
)

func TestStore(t *testing.T) {
	ctx := context.Background()
	store, err := db.Open(t.TempDir())
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer store.Close()

	runID := "run-12345"
	if err := store.RecordStart(ctx, runID, "testdb", "postgres"); err != nil {
		t.Fatalf("failed to record start: %v", err)
	}

	run, err := store.GetRun(ctx, runID)
	if err != nil {
		t.Fatalf("failed to get run: %v", err)
	}
	if run.Status != "running" {
		t.Errorf("expected running, got %s", run.Status)
	}

	dests := []string{"local", "s3"}
	if err := store.RecordFinish(ctx, runID, "success", 1024*1024, 5*time.Second, dests, "", "backup completed"); err != nil {
		t.Fatalf("failed to record finish: %v", err)
	}

	run, err = store.GetRun(ctx, runID)
	if err != nil {
		t.Fatalf("failed to get run after finish: %v", err)
	}
	if run.Status != "success" {
		t.Errorf("expected success, got %s", run.Status)
	}
	if run.SizeBytes != 1024*1024 {
		t.Errorf("expected 1MB, got %d", run.SizeBytes)
	}
	if len(run.Destinations) != 2 {
		t.Errorf("expected 2 destinations, got %d", len(run.Destinations))
	}

	stats, err := store.GetSummaryStats(ctx)
	if err != nil {
		t.Fatalf("failed to get summary stats: %v", err)
	}
	if stats.TotalBackups != 1 || stats.SuccessfulCount != 1 || stats.FailedCount != 0 {
		t.Errorf("unexpected stats: %+v", stats)
	}
	if stats.LastBackupAt == nil {
		t.Errorf("expected LastBackupAt to be non-nil, got nil")
	}

	// Verify GetRecentRuns does NOT return heavy logs
	recentRuns, total, err := store.GetRecentRuns(ctx, 10, 0, "")
	if err != nil {
		t.Fatalf("failed to get recent runs: %v", err)
	}
	if total != 1 || len(recentRuns) != 1 {
		t.Fatalf("expected 1 recent run, got total=%d len=%d", total, len(recentRuns))
	}
	if recentRuns[0].Logs != "" {
		t.Errorf("expected GetRecentRuns to omit logs, got %q", recentRuns[0].Logs)
	}

	// Verify GetRunLogs fetches logs
	logs, err := store.GetRunLogs(ctx, runID)
	if err != nil {
		t.Fatalf("failed to get run logs: %v", err)
	}
	if logs != "backup completed" {
		t.Errorf("expected 'backup completed', got %q", logs)
	}

	// Verify PruneRuns
	// Cutoff in the past: nothing pruned
	pruned, err := store.PruneRuns(ctx, time.Now().UTC().Add(-1*time.Hour))
	if err != nil {
		t.Fatalf("failed to prune runs: %v", err)
	}
	if pruned != 0 {
		t.Errorf("expected 0 pruned, got %d", pruned)
	}

	// Cutoff in the future: run is pruned
	pruned, err = store.PruneRuns(ctx, time.Now().UTC().Add(1*time.Hour))
	if err != nil {
		t.Fatalf("failed to prune runs: %v", err)
	}
	if pruned != 1 {
		t.Errorf("expected 1 pruned, got %d", pruned)
	}
}
