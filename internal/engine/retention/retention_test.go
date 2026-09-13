package retention_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/blesswinsamuel/arkbase/internal/config"
	"github.com/blesswinsamuel/arkbase/internal/engine/retention"
	"github.com/blesswinsamuel/arkbase/internal/engine/storage"
)

func TestEvaluateRetention(t *testing.T) {
	now := time.Date(2026, 9, 14, 12, 0, 0, 0, time.UTC)

	// Create 10 items spaced 1 day apart
	var items []storage.FileItem
	for i := 0; i < 10; i++ {
		items = append(items, storage.FileItem{
			Path:    fmt.Sprintf("backup-day-%d.sql.gz", i),
			Size:    1024,
			ModTime: now.Add(-time.Duration(i) * 24 * time.Hour),
		})
	}

	policy := &config.RetentionPolicy{
		KeepLast: 3,
		Daily:    5,
	}

	keep, prune := retention.EvaluateRetention(policy, items, now)

	// Should keep days 0, 1, 2 (keepLast 3), and days 3, 4, 5 (daily within 5 days)
	// Days 6, 7, 8, 9 should be pruned
	if len(keep) != 6 {
		t.Errorf("expected 6 kept items, got %d", len(keep))
	}
	if len(prune) != 4 {
		t.Errorf("expected 4 pruned items, got %d", len(prune))
	}
}
