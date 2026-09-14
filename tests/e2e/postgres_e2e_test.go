package e2e_test

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/blesswinsamuel/arkbase/internal/config"
	"github.com/blesswinsamuel/arkbase/internal/db"
	"github.com/blesswinsamuel/arkbase/internal/engine"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestPostgresBackupAndRestoreE2E(t *testing.T) {
	ctx := context.Background()

	// 1. Start source PostgreSQL container
	sourceContainer, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("sourcedb"),
		postgres.WithUsername("arkuser"),
		postgres.WithPassword("arkpass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		t.Skipf("skipping E2E test: docker or testcontainers not available: %v", err)
		return
	}
	defer func() {
		_ = sourceContainer.Terminate(ctx)
	}()

	sourceHost, err := sourceContainer.Host(ctx)
	if err != nil {
		t.Fatalf("failed to get source host: %v", err)
	}
	sourcePort, err := sourceContainer.MappedPort(ctx, "5432")
	if err != nil {
		t.Fatalf("failed to get source port: %v", err)
	}

	sourceConnStr := fmt.Sprintf("postgres://arkuser:arkpass@%s:%s/sourcedb?sslmode=disable", sourceHost, sourcePort.Port())
	sourceDB, err := sql.Open("postgres", sourceConnStr)
	if err == nil {
		defer sourceDB.Close()
	}

	// 2. Start target PostgreSQL container (for testing restore)
	targetContainer, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("targetdb"),
		postgres.WithUsername("arkuser"),
		postgres.WithPassword("arkpass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		t.Fatalf("failed to start target container: %v", err)
	}
	defer func() {
		_ = targetContainer.Terminate(ctx)
	}()

	targetHost, err := targetContainer.Host(ctx)
	if err != nil {
		t.Fatalf("failed to get target host: %v", err)
	}
	targetPort, err := targetContainer.MappedPort(ctx, "5432")
	if err != nil {
		t.Fatalf("failed to get target port: %v", err)
	}

	// 3. Setup arkbase config
	backupDir := t.TempDir()
	sourceP, _ := strconv.Atoi(sourcePort.Port())
	targetP, _ := strconv.Atoi(targetPort.Port())

	cfg := &config.Config{
		Server: config.ServerConfig{Port: 8080, DataDir: t.TempDir()},
		Destinations: map[string]config.DestinationConfig{
			"local_encrypted": {
				Type: config.DestinationTypeFilesystem,
				Path: filepath.Join(backupDir, "{{ .Database }}-{{ .Timestamp }}.sql.gz.enc"),
				Encryption: &config.EncryptionConfig{
					Enabled:    true,
					Passphrase: "e2e-super-secret-key",
				},
			},
		},
		Databases: map[string]config.DatabaseConfig{
			"sourcedb": {
				Engine:       "postgres",
				Host:         sourceHost,
				Port:         sourceP,
				Username:     "arkuser",
				Password:     "arkpass",
				Database:     "sourcedb",
				SSLMode:      "disable",
				Destinations: []string{"local_encrypted"},
			},
			"targetdb": {
				Engine:   "postgres",
				Host:     targetHost,
				Port:     targetP,
				Username: "arkuser",
				Password: "arkpass",
				Database: "targetdb",
				SSLMode:  "disable",
			},
		},
	}

	store, err := db.Open(cfg.Server.DataDir)
	if err != nil {
		t.Fatalf("failed to init db: %v", err)
	}
	defer store.Close()

	runner, err := engine.NewRunner(cfg, store)
	if err != nil {
		t.Fatalf("failed to init runner: %v", err)
	}

	// 4. Run backup
	run, err := runner.RunBackup(ctx, "sourcedb")
	if err != nil {
		t.Fatalf("backup failed: %v", err)
	}
	if run.Status != "success" {
		t.Fatalf("expected backup success, got %s: %s", run.Status, run.ErrorMessage)
	}

	// 5. Verify encrypted backup file exists on disk
	files, err := os.ReadDir(backupDir)
	if err != nil || len(files) == 0 {
		t.Fatalf("no backup files found in %s", backupDir)
	}
	backupFilePath := filepath.Join(backupDir, files[0].Name())

	// 6. Restore backup into targetdb
	err = runner.RestoreBackup(ctx, "targetdb", "local_encrypted", backupFilePath, os.Stdout)
	if err != nil {
		t.Fatalf("restore into targetdb failed: %v", err)
	}

	t.Logf("E2E backup and restore verified successfully!")
}
