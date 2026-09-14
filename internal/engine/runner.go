package engine

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/blesswinsamuel/arkbase/internal/config"
	"github.com/blesswinsamuel/arkbase/internal/db"
	"github.com/blesswinsamuel/arkbase/internal/engine/drivers"
	"github.com/blesswinsamuel/arkbase/internal/engine/encryption"
	"github.com/blesswinsamuel/arkbase/internal/engine/retention"
	"github.com/blesswinsamuel/arkbase/internal/engine/storage"
	"github.com/blesswinsamuel/arkbase/internal/notify"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

type Runner struct {
	cfg          *config.Config
	store        *db.Store
	destinations map[string]storage.StorageTarget
	runningMu    sync.Mutex
	runningJobs  map[string]bool
}

func NewRunner(cfg *config.Config, store *db.Store) (*Runner, error) {
	targets := make(map[string]storage.StorageTarget)
	for name, destCfg := range cfg.Destinations {
		target, err := storage.NewStorageTarget(destCfg)
		if err != nil {
			return nil, fmt.Errorf("init storage %s: %w", name, err)
		}
		targets[name] = target
	}

	return &Runner{
		cfg:          cfg,
		store:        store,
		destinations: targets,
		runningJobs:  make(map[string]bool),
	}, nil
}

func (r *Runner) IsRunning(dbName string) bool {
	r.runningMu.Lock()
	defer r.runningMu.Unlock()
	return r.runningJobs[dbName]
}

func (r *Runner) RunBackup(ctx context.Context, dbName string) (*db.Run, error) {
	dbCfg, ok := r.cfg.Databases[dbName]
	if !ok {
		return nil, fmt.Errorf("database %q not found in config", dbName)
	}

	r.runningMu.Lock()
	if r.runningJobs[dbName] {
		r.runningMu.Unlock()
		return nil, fmt.Errorf("backup for database %q is already in progress", dbName)
	}
	r.runningJobs[dbName] = true
	r.runningMu.Unlock()

	defer func() {
		r.runningMu.Lock()
		delete(r.runningJobs, dbName)
		r.runningMu.Unlock()
	}()

	runID := uuid.New().String()
	startTime := time.Now().UTC()
	var logBuf bytes.Buffer

	log.Info().Str("run_id", runID).Str("database", dbName).Msg("starting backup run")
	fmt.Fprintf(&logBuf, "[%s] Starting backup for %s (engine: %s)\n", startTime.Format(time.RFC3339), dbName, dbCfg.Engine)

	if err := r.store.RecordStart(ctx, runID, dbName, dbCfg.Engine); err != nil {
		log.Error().Err(err).Msg("failed to record backup start in store")
	}

	driver, err := drivers.GetDriver(dbCfg.Engine)
	if err != nil {
		r.recordFailure(ctx, runID, dbName, dbCfg.Engine, 0, time.Since(startTime), nil, err.Error(), logBuf.String())
		return nil, err
	}

	// Create temp directory for staging the dump
	tempDir := filepath.Join(r.cfg.Server.DataDir, "tmp")
	_ = os.MkdirAll(tempDir, 0755)

	dumpRawFile := filepath.Join(tempDir, fmt.Sprintf("%s-%s.raw", dbName, runID))
	defer os.Remove(dumpRawFile)

	rawF, err := os.Create(dumpRawFile)
	if err != nil {
		r.recordFailure(ctx, runID, dbName, dbCfg.Engine, 0, time.Since(startTime), nil, err.Error(), logBuf.String())
		return nil, err
	}

	// Execute dump
	dumpErr := driver.Dump(ctx, dbCfg, rawF, &logBuf)
	_ = rawF.Close()

	if dumpErr != nil {
		errMsg := fmt.Sprintf("dump error: %v", dumpErr)
		fmt.Fprintf(&logBuf, "[%s] ERROR: %s\n", time.Now().UTC().Format(time.RFC3339), errMsg)
		r.recordFailure(ctx, runID, dbName, dbCfg.Engine, 0, time.Since(startTime), nil, errMsg, logBuf.String())
		return nil, dumpErr
	}

	rawFi, err := os.Stat(dumpRawFile)
	if err != nil {
		r.recordFailure(ctx, runID, dbName, dbCfg.Engine, 0, time.Since(startTime), nil, err.Error(), logBuf.String())
		return nil, err
	}
	rawSize := rawFi.Size()
	fmt.Fprintf(&logBuf, "[%s] Dump succeeded. Size: %d bytes (%.2f MB)\n", time.Now().UTC().Format(time.RFC3339), rawSize, float64(rawSize)/(1024*1024))

	var successfulDests []string
	timestampStr := startTime.Format("20060102-150405")

	// Upload to each configured destination
	for _, destName := range dbCfg.Destinations {
		destCfg, ok := r.cfg.Destinations[destName]
		if !ok {
			fmt.Fprintf(&logBuf, "[%s] WARN: Destination %q not found in config\n", time.Now().UTC().Format(time.RFC3339), destName)
			continue
		}

		target := r.destinations[destName]
		targetPath, err := interpolatePath(destCfg.Path, dbName, dbCfg.Engine, timestampStr)
		if err != nil {
			fmt.Fprintf(&logBuf, "[%s] ERROR: Path template for %s: %v\n", time.Now().UTC().Format(time.RFC3339), destName, err)
			continue
		}

		// Check if encryption is required for this destination
		uploadFile := dumpRawFile
		uploadSize := rawSize
		encryptedFile := ""

		if destCfg.Encryption != nil && destCfg.Encryption.Enabled && destCfg.Encryption.Passphrase != "" {
			encryptedFile = filepath.Join(tempDir, fmt.Sprintf("%s-%s-%s.enc", dbName, destName, runID))
			defer os.Remove(encryptedFile)

			encF, err := os.Create(encryptedFile)
			if err != nil {
				fmt.Fprintf(&logBuf, "[%s] ERROR: create encryption staging file: %v\n", time.Now().UTC().Format(time.RFC3339), err)
				continue
			}

			srcF, err := os.Open(dumpRawFile)
			if err != nil {
				_ = encF.Close()
				fmt.Fprintf(&logBuf, "[%s] ERROR: open raw dump for encryption: %v\n", time.Now().UTC().Format(time.RFC3339), err)
				continue
			}

			encErr := encryption.EncryptStream(destCfg.Encryption.Passphrase, srcF, encF)
			_ = srcF.Close()
			_ = encF.Close()

			if encErr != nil {
				fmt.Fprintf(&logBuf, "[%s] ERROR: encryption failed for %s: %v\n", time.Now().UTC().Format(time.RFC3339), destName, encErr)
				continue
			}

			encFi, _ := os.Stat(encryptedFile)
			uploadFile = encryptedFile
			uploadSize = encFi.Size()
			fmt.Fprintf(&logBuf, "[%s] Encrypted dump with AES-256-GCM. Encrypted size: %d bytes\n", time.Now().UTC().Format(time.RFC3339), uploadSize)
		}

		// Upload to target
		upF, err := os.Open(uploadFile)
		if err != nil {
			fmt.Fprintf(&logBuf, "[%s] ERROR: open file for upload: %v\n", time.Now().UTC().Format(time.RFC3339), err)
			continue
		}

		fmt.Fprintf(&logBuf, "[%s] Uploading to destination %q (%s) at %s...\n", time.Now().UTC().Format(time.RFC3339), destName, destCfg.Type, targetPath)
		saveErr := target.Save(ctx, targetPath, upF, uploadSize)
		_ = upF.Close()

		if saveErr != nil {
			fmt.Fprintf(&logBuf, "[%s] ERROR: failed uploading to %s: %v\n", time.Now().UTC().Format(time.RFC3339), destName, saveErr)
			continue
		}

		fmt.Fprintf(&logBuf, "[%s] Upload to %q succeeded\n", time.Now().UTC().Format(time.RFC3339), destName)
		successfulDests = append(successfulDests, destName)

		// Prune according to retention policy
		if destCfg.Retention != nil {
			prefix := filepath.Dir(targetPath)
			items, listErr := target.List(ctx, prefix)
			if listErr == nil {
				_, pruneItems := retention.EvaluateRetention(destCfg.Retention, items, time.Now().UTC())
				for _, item := range pruneItems {
					if item.Path != targetPath {
						fmt.Fprintf(&logBuf, "[%s] Pruning old snapshot: %s\n", time.Now().UTC().Format(time.RFC3339), item.Path)
						_ = target.Delete(ctx, item.Path)
					}
				}
			}
		}
	}

	duration := time.Since(startTime)
	if len(successfulDests) == 0 && len(dbCfg.Destinations) > 0 {
		errMsg := "all destination uploads failed"
		r.recordFailure(ctx, runID, dbName, dbCfg.Engine, rawSize, duration, successfulDests, errMsg, logBuf.String())
		return nil, fmt.Errorf("%s", errMsg)
	}

	// Success!
	fmt.Fprintf(&logBuf, "[%s] Backup run finished successfully in %s\n", time.Now().UTC().Format(time.RFC3339), duration.Round(time.Millisecond))
	_ = r.store.RecordFinish(ctx, runID, "success", rawSize, duration, successfulDests, "", logBuf.String())

	MetricBackupCount.WithLabelValues(dbName, "success").Inc()
	MetricBackupDuration.WithLabelValues(dbName).Observe(duration.Seconds())
	MetricLastSuccessTimestamp.WithLabelValues(dbName).Set(float64(time.Now().Unix()))
	MetricBackupSizeBytes.WithLabelValues(dbName).Set(float64(rawSize))

	notify.DispatchNotifications(ctx, r.cfg.Notifications, notify.EventPayload{
		Event:        "success",
		DatabaseName: dbName,
		Engine:       dbCfg.Engine,
		SizeBytes:    rawSize,
		Duration:     duration,
		Destinations: successfulDests,
		Timestamp:    time.Now().UTC(),
	})

	return r.store.GetRun(ctx, runID)
}

func (r *Runner) recordFailure(ctx context.Context, runID, dbName, engine string, size int64, duration time.Duration, dests []string, errMsg, logs string) {
	log.Error().Str("run_id", runID).Str("database", dbName).Str("error", errMsg).Msg("backup run failed")
	_ = r.store.RecordFinish(ctx, runID, "failed", size, duration, dests, errMsg, logs)

	MetricBackupCount.WithLabelValues(dbName, "failure").Inc()

	notify.DispatchNotifications(ctx, r.cfg.Notifications, notify.EventPayload{
		Event:        "failure",
		DatabaseName: dbName,
		Engine:       engine,
		SizeBytes:    size,
		Duration:     duration,
		Destinations: dests,
		Error:        errMsg,
		Timestamp:    time.Now().UTC(),
	})
}

func interpolatePath(pathTemplate, dbName, engine, timestamp string) (string, error) {
	tmpl, err := template.New("path").Parse(pathTemplate)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	data := map[string]string{
		"Database":  dbName,
		"Engine":    engine,
		"Timestamp": timestamp,
	}
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// StorageBackupItem represents a physical backup file stored in a destination target
type StorageBackupItem struct {
	Database        string    `json:"database"`
	Destination     string    `json:"destination"`
	DestinationType string    `json:"destination_type"`
	Path            string    `json:"path"`
	Filename        string    `json:"filename"`
	SizeBytes       int64     `json:"size_bytes"`
	ModTime         time.Time `json:"mod_time"`
	Encrypted       bool      `json:"encrypted"`
}

// ListDatabaseBackups queries the storage targets configured for a database and returns matching backup files
func (r *Runner) ListDatabaseBackups(ctx context.Context, dbName, destinationFilter string) ([]StorageBackupItem, error) {
	dbCfg, ok := r.cfg.Databases[dbName]
	if !ok {
		return nil, fmt.Errorf("database %q not found in config", dbName)
	}

	var allItems []StorageBackupItem

	for _, destName := range dbCfg.Destinations {
		if destinationFilter != "" && destName != destinationFilter {
			continue
		}

		destCfg, ok := r.cfg.Destinations[destName]
		if !ok {
			continue
		}

		target, ok := r.destinations[destName]
		if !ok {
			continue
		}

		// Interpolate with placeholder to find the directory and filename pattern
		interpolated, err := interpolatePath(destCfg.Path, dbName, dbCfg.Engine, "__TS__")
		if err != nil {
			log.Warn().Err(err).Str("destination", destName).Msg("failed to interpolate path for backup list")
			continue
		}

		dir := filepath.Dir(interpolated)
		baseFilename := filepath.Base(interpolated)
		var prefixMatch, suffixMatch string
		if strings.Contains(baseFilename, "__TS__") {
			parts := strings.SplitN(baseFilename, "__TS__", 2)
			prefixMatch = parts[0]
			suffixMatch = parts[1]
		}

		items, err := target.List(ctx, dir)
		if err != nil {
			log.Warn().Err(err).Str("destination", destName).Msg("failed to list destination files")
			continue
		}

		isEncrypted := destCfg.Encryption != nil && destCfg.Encryption.Enabled

		for _, item := range items {
			fname := filepath.Base(item.Path)
			if prefixMatch != "" && !strings.HasPrefix(fname, prefixMatch) {
				continue
			}
			if suffixMatch != "" && !strings.HasSuffix(fname, suffixMatch) {
				continue
			}

			allItems = append(allItems, StorageBackupItem{
				Database:        dbName,
				Destination:     destName,
				DestinationType: string(destCfg.Type),
				Path:            item.Path,
				Filename:        fname,
				SizeBytes:       item.Size,
				ModTime:         item.ModTime,
				Encrypted:       isEncrypted,
			})
		}
	}

	// Sort newest first
	sort.Slice(allItems, func(i, j int) bool {
		return allItems[i].ModTime.After(allItems[j].ModTime)
	})

	return allItems, nil
}

// RestoreBackup restores a backup file from a storage destination into a target database
func (r *Runner) RestoreBackup(ctx context.Context, dbName, destName, backupPath string, logWriter io.Writer) error {
	dbCfg, ok := r.cfg.Databases[dbName]
	if !ok {
		return fmt.Errorf("database %q not found", dbName)
	}
	destCfg, ok := r.cfg.Destinations[destName]
	if !ok {
		return fmt.Errorf("destination %q not found", destName)
	}
	target, ok := r.destinations[destName]
	if !ok {
		return fmt.Errorf("target handler %q not found", destName)
	}

	driver, err := drivers.GetDriver(dbCfg.Engine)
	if err != nil {
		return err
	}

	r.runningMu.Lock()
	if r.runningJobs[dbName] {
		r.runningMu.Unlock()
		return fmt.Errorf("database %q has an active backup or restore in progress", dbName)
	}
	r.runningJobs[dbName] = true
	r.runningMu.Unlock()

	defer func() {
		r.runningMu.Lock()
		delete(r.runningJobs, dbName)
		r.runningMu.Unlock()
	}()

	startTime := time.Now().UTC()
	if logWriter != nil {
		fmt.Fprintf(logWriter, "[%s] Starting restore for database %q from destination %q (%s)\n", startTime.Format(time.RFC3339), dbName, destName, backupPath)
	}

	reader, err := target.Open(ctx, backupPath)
	if err != nil {
		return fmt.Errorf("open backup file %s: %w", backupPath, err)
	}
	defer reader.Close()

	var stream io.Reader = reader

	// If encrypted, decrypt on the fly
	if destCfg.Encryption != nil && destCfg.Encryption.Enabled && destCfg.Encryption.Passphrase != "" {
		if logWriter != nil {
			fmt.Fprintf(logWriter, "[%s] Decrypting AES-256-GCM backup stream...\n", time.Now().UTC().Format(time.RFC3339))
		}
		pr, pw := io.Pipe()
		go func() {
			decErr := encryption.DecryptStream(destCfg.Encryption.Passphrase, reader, pw)
			_ = pw.CloseWithError(decErr)
		}()
		stream = pr
	}

	if logWriter != nil {
		fmt.Fprintf(logWriter, "[%s] Applying restore stream using %s driver...\n", time.Now().UTC().Format(time.RFC3339), dbCfg.Engine)
	}

	restoreErr := driver.Restore(ctx, dbCfg, stream, logWriter)
	duration := time.Since(startTime)

	if restoreErr != nil {
		if logWriter != nil {
			fmt.Fprintf(logWriter, "[%s] ERROR: Restore failed after %s: %v\n", time.Now().UTC().Format(time.RFC3339), duration.Round(time.Millisecond), restoreErr)
		}
		return restoreErr
	}

	if logWriter != nil {
		fmt.Fprintf(logWriter, "[%s] Restore completed successfully in %s\n", time.Now().UTC().Format(time.RFC3339), duration.Round(time.Millisecond))
	}
	return nil
}

// PruneHistory removes backup_runs records older than HistoryRetentionDays.
func (r *Runner) PruneHistory(ctx context.Context) (int64, error) {
	days := r.cfg.Server.HistoryRetentionDays
	if days <= 0 {
		return 0, nil
	}
	cutoff := time.Now().UTC().Add(-time.Duration(days) * 24 * time.Hour)
	pruned, err := r.store.PruneRuns(ctx, cutoff)
	if err != nil {
		log.Error().Err(err).Int("days", days).Msg("failed to prune old backup history")
		return 0, err
	}
	if pruned > 0 {
		log.Info().Int64("pruned_count", pruned).Int("retention_days", days).Msg("pruned old backup runs from history")
	}
	return pruned, nil
}

