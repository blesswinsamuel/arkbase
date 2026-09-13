package config_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/blesswinsamuel/arkbase/internal/config"
)

func TestLoadConfig(t *testing.T) {
	os.Setenv("TEST_PG_PASS", "secret123")
	os.Setenv("TEST_S3_KEY", "minioadmin")

	yamlContent := `
server:
  port: 9090
  data_dir: /tmp/arkbase-test
  auth:
    enabled: true
    username: myadmin
    password: "${TEST_PG_PASS}"

destinations:
  local:
    type: filesystem
    path: /backups/{{ .Database }}.sql.gz
    retention:
      keep_last: 5
      daily: 7
  s3_backup:
    type: s3
    endpoint: minio:9000
    bucket: test-bucket
    access_key: "${TEST_S3_KEY}"
    secret_key: "${TEST_PG_PASS}"
    path: backups/{{ .Database }}.enc
    encryption:
      enabled: true
      passphrase: "${TEST_PG_PASS}"

databases:
  users_db:
    engine: postgres
    host: pg.internal
    port: 5432
    username: postgres
    password: "${TEST_PG_PASS}"
    database: users
    schedule: "0 */4 * * *"
    destinations:
      - local
      - s3_backup

notifications:
  - name: test_hook
    type: webhook
    url: "https://example.com/ping"
    on_events: [success, failure]
`
	tmpFile := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(tmpFile, []byte(yamlContent), 0600); err != nil {
		t.Fatalf("failed to write tmp config: %v", err)
	}

	cfg, err := config.LoadConfig(tmpFile)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if cfg.Server.Port != 9090 {
		t.Errorf("expected port 9090, got %d", cfg.Server.Port)
	}
	if cfg.Server.Auth.Password != "secret123" {
		t.Errorf("expected expanded password 'secret123', got '%s'", cfg.Server.Auth.Password)
	}
	if len(cfg.Destinations) != 2 {
		t.Errorf("expected 2 destinations, got %d", len(cfg.Destinations))
	}
	db, ok := cfg.Databases["users_db"]
	if !ok {
		t.Fatalf("expected users_db in databases")
	}
	if db.Password != "secret123" {
		t.Errorf("expected users_db password 'secret123', got '%s'", db.Password)
	}
	if db.Timeout != 30*time.Minute {
		t.Errorf("expected default timeout 30m, got %v", db.Timeout)
	}
}
