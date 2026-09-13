package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/blesswinsamuel/arkbase/internal/api"
	"github.com/blesswinsamuel/arkbase/internal/config"
	"github.com/blesswinsamuel/arkbase/internal/db"
	"github.com/blesswinsamuel/arkbase/internal/engine"
	"github.com/go-chi/chi/v5"
)

func TestAPI(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{Port: 8080, DataDir: t.TempDir()},
		Databases: map[string]config.DatabaseConfig{
			"testdb": {Engine: "postgres", Host: "localhost", Database: "testdb"},
		},
		Destinations: map[string]config.DestinationConfig{
			"local": {Type: config.DestinationTypeFilesystem, Path: "/tmp/test.sql.gz"},
		},
	}

	store, err := db.Open(cfg.Server.DataDir)
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer store.Close()

	runner, err := engine.NewRunner(cfg, store)
	if err != nil {
		t.Fatalf("failed to create runner: %v", err)
	}

	scheduler, err := engine.NewScheduler(cfg, runner)
	if err != nil {
		t.Fatalf("failed to create scheduler: %v", err)
	}

	srv := api.NewServer(cfg, store, runner, scheduler, "1.0.0-test")
	r := chi.NewRouter()
	srv.SetupRouter(r)

	// 1. Test GET /api/v1/status
	req := httptest.NewRequest(http.MethodGet, "/api/v1/status", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var statusResp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &statusResp); err != nil {
		t.Fatalf("failed to unmarshal status: %v", err)
	}
	if statusResp["status"] != "healthy" {
		t.Errorf("expected healthy status, got %v", statusResp["status"])
	}

	// 2. Test GET /openapi.json (Verify OpenAPI 3.1 generation)
	reqOpenAPI := httptest.NewRequest(http.MethodGet, "/openapi.json", nil)
	recOpenAPI := httptest.NewRecorder()
	r.ServeHTTP(recOpenAPI, reqOpenAPI)

	if recOpenAPI.Code != http.StatusOK {
		t.Fatalf("expected 200 for openapi.json, got %d", recOpenAPI.Code)
	}
	var openAPISpec map[string]any
	if err := json.Unmarshal(recOpenAPI.Body.Bytes(), &openAPISpec); err != nil {
		t.Fatalf("invalid json for openapi: %v", err)
	}
	if openAPISpec["openapi"] == nil {
		t.Errorf("missing openapi version in spec")
	}

	// 3. Test GET /api/v1/databases
	reqDBs := httptest.NewRequest(http.MethodGet, "/api/v1/databases", nil)
	recDBs := httptest.NewRecorder()
	r.ServeHTTP(recDBs, reqDBs)

	if recDBs.Code != http.StatusOK {
		t.Fatalf("expected 200 for databases, got %d", recDBs.Code)
	}
	var dbs []any
	if err := json.Unmarshal(recDBs.Body.Bytes(), &dbs); err != nil {
		t.Fatalf("failed to unmarshal databases: %v", err)
	}
	if len(dbs) != 1 {
		t.Errorf("expected 1 database, got %d", len(dbs))
	}

	// 4. Test GET /api/v1/databases/testdb/backups
	reqBackups := httptest.NewRequest(http.MethodGet, "/api/v1/databases/testdb/backups", nil)
	recBackups := httptest.NewRecorder()
	r.ServeHTTP(recBackups, reqBackups)

	if recBackups.Code != http.StatusOK {
		t.Fatalf("expected 200 for backups, got %d: %s", recBackups.Code, recBackups.Body.String())
	}
	var backupsResp struct {
		Backups []any `json:"backups"`
	}
	if err := json.Unmarshal(recBackups.Body.Bytes(), &backupsResp); err != nil {
		t.Fatalf("failed to unmarshal backups response: %v", err)
	}

	// 5. Test 404 for nonexistent database
	req404 := httptest.NewRequest(http.MethodGet, "/api/v1/databases/nonexistent/backups", nil)
	rec404 := httptest.NewRecorder()
	r.ServeHTTP(rec404, req404)
	if rec404.Code != http.StatusNotFound {
		t.Errorf("expected 404 for nonexistent db backups, got %d", rec404.Code)
	}

	// 6. Test 400 for restore without destination or path
	reqRestoreBad := httptest.NewRequest(http.MethodPost, "/api/v1/databases/testdb/restore", nil)
	recRestoreBad := httptest.NewRecorder()
	r.ServeHTTP(recRestoreBad, reqRestoreBad)
	if recRestoreBad.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for restore with missing body, got %d", recRestoreBad.Code)
	}
}
