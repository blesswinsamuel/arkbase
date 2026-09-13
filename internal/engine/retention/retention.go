package retention

import (
	"sort"
	"time"

	"github.com/blesswinsamuel/arkbase/internal/config"
	"github.com/blesswinsamuel/arkbase/internal/engine/storage"
)

// EvaluateRetention determines which files should be kept and which should be pruned.
func EvaluateRetention(policy *config.RetentionPolicy, items []storage.FileItem, now time.Time) (keep []storage.FileItem, prune []storage.FileItem) {
	if policy == nil || len(items) == 0 {
		return items, nil
	}

	// Sort items descending by modification time (newest first)
	sorted := make([]storage.FileItem, len(items))
	copy(sorted, items)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].ModTime.After(sorted[j].ModTime)
	})

	keepSet := make(map[string]bool)

	// 1. Keep the most recent N backups
	keepLast := policy.KeepLast
	if keepLast <= 0 {
		keepLast = 1 // At least keep the newest 1
	}
	for i := 0; i < len(sorted) && i < keepLast; i++ {
		keepSet[sorted[i].Path] = true
	}

	// Helper for slotting (hourly, daily, weekly, monthly, yearly)
	seenHourly := make(map[string]bool)
	seenDaily := make(map[string]bool)
	seenWeekly := make(map[string]bool)
	seenMonthly := make(map[string]bool)
	seenYearly := make(map[string]bool)

	for _, item := range sorted {
		if keepSet[item.Path] {
			continue
		}

		age := now.Sub(item.ModTime)

		// Hourly retention
		if policy.Hourly > 0 && age <= time.Duration(policy.Hourly)*time.Hour {
			hourKey := item.ModTime.Format("2006-01-02-15")
			if !seenHourly[hourKey] {
				seenHourly[hourKey] = true
				keepSet[item.Path] = true
				continue
			}
		}

		// Daily retention
		if policy.Daily > 0 && age <= time.Duration(policy.Daily)*24*time.Hour {
			dayKey := item.ModTime.Format("2006-01-02")
			if !seenDaily[dayKey] {
				seenDaily[dayKey] = true
				keepSet[item.Path] = true
				continue
			}
		}

		// Weekly retention
		if policy.Weekly > 0 && age <= time.Duration(policy.Weekly)*7*24*time.Hour {
			year, week := item.ModTime.ISOWeek()
			weekKey := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC).Format("2006") + "-" + string(rune(week))
			if !seenWeekly[weekKey] {
				seenWeekly[weekKey] = true
				keepSet[item.Path] = true
				continue
			}
		}

		// Monthly retention
		if policy.Monthly > 0 && age <= time.Duration(policy.Monthly)*30*24*time.Hour {
			monthKey := item.ModTime.Format("2006-01")
			if !seenMonthly[monthKey] {
				seenMonthly[monthKey] = true
				keepSet[item.Path] = true
				continue
			}
		}

		// Yearly retention
		if policy.Yearly > 0 && age <= time.Duration(policy.Yearly)*365*24*time.Hour {
			yearKey := item.ModTime.Format("2006")
			if !seenYearly[yearKey] {
				seenYearly[yearKey] = true
				keepSet[item.Path] = true
				continue
			}
		}
	}

	for _, item := range sorted {
		if keepSet[item.Path] {
			keep = append(keep, item)
		} else {
			prune = append(prune, item)
		}
	}

	return keep, prune
}
