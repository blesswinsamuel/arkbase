package engine

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	MetricBackupCount = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "arkbase",
		Name:      "backup_count_total",
		Help:      "Total number of backup executions",
	}, []string{"database", "status"})

	MetricBackupDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "arkbase",
		Name:      "backup_duration_seconds",
		Help:      "Duration of backup executions in seconds",
		Buckets:   prometheus.DefBuckets,
	}, []string{"database"})

	MetricLastSuccessTimestamp = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "arkbase",
		Name:      "last_successful_backup_timestamp",
		Help:      "Timestamp of the last successful backup in unix seconds",
	}, []string{"database"})

	MetricBackupSizeBytes = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "arkbase",
		Name:      "backup_size_bytes",
		Help:      "Size of the latest backup in bytes",
	}, []string{"database"})
)
