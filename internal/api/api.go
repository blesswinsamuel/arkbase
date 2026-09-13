package api

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/blesswinsamuel/arkbase/internal/config"
	"github.com/blesswinsamuel/arkbase/internal/db"
	"github.com/blesswinsamuel/arkbase/internal/engine"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog/log"
)

type Server struct {
	cfg       *config.Config
	store     *db.Store
	runner    *engine.Runner
	scheduler *engine.Scheduler
	startTime time.Time
	version   string
}

func NewServer(cfg *config.Config, store *db.Store, runner *engine.Runner, scheduler *engine.Scheduler, version string) *Server {
	return &Server{
		cfg:       cfg,
		store:     store,
		runner:    runner,
		scheduler: scheduler,
		startTime: time.Now().UTC(),
		version:   version,
	}
}

func (s *Server) SetupRouter(r chi.Router) huma.API {
	// Expose prometheus metrics
	r.Handle("/metrics", promhttp.Handler())

	humaConfig := huma.DefaultConfig("arkbase API", s.version)
	humaConfig.DocsPath = "/docs"
	humaConfig.DocsRenderer = "scalar"
	humaConfig.OpenAPIPath = "/openapi"

	api := humachi.New(r, humaConfig)

	s.registerRoutes(api)
	return api
}

// DTOs

type StatusOutput struct {
	Body struct {
		Status           string `json:"status" example:"healthy" doc:"Overall system health"`
		Version          string `json:"version" example:"1.0.0" doc:"arkbase version"`
		UptimeSeconds    int64  `json:"uptime_seconds" example:"3600" doc:"Daemon uptime in seconds"`
		DatabaseCount    int    `json:"database_count" example:"5" doc:"Total configured databases"`
		DestinationCount int    `json:"destination_count" example:"2" doc:"Total configured storage destinations"`
	}
}

type DatabaseInfo struct {
	Name         string     `json:"name" example:"production_pg"`
	Engine       string     `json:"engine" example:"postgres"`
	Host         string     `json:"host" example:"pg-cluster.internal"`
	Port         int        `json:"port" example:"5432"`
	Database     string     `json:"database" example:"app_db"`
	Schedule     string     `json:"schedule" example:"0 */6 * * *"`
	Destinations []string   `json:"destinations" example:"[\"local_storage\", \"offsite_s3\"]"`
	IsRunning    bool       `json:"is_running" example:"false"`
	NextRunAt    *time.Time `json:"next_run_at,omitempty"`
	LastRun      *db.Run    `json:"last_run,omitempty"`
}

type DatabasesOutput struct {
	Body []DatabaseInfo
}

type TriggerBackupInput struct {
	ID string `path:"id" doc:"Name of the database to backup"`
}

type TriggerBackupOutput struct {
	Body struct {
		Message string `json:"message" example:"Backup triggered"`
		RunID   string `json:"run_id,omitempty"`
	}
}

type ListBackupsInput struct {
	ID          string `path:"id" doc:"Name of the database"`
	Destination string `query:"destination" doc:"Optional filter by storage destination name"`
}

type ListBackupsOutput struct {
	Body struct {
		Backups []engine.StorageBackupItem `json:"backups"`
	}
}

type RestoreBackupInput struct {
	ID   string `path:"id" doc:"Name of the database to restore"`
	Body struct {
		Destination string `json:"destination" doc:"Name of storage destination"`
		BackupPath  string `json:"backup_path" doc:"Path of backup file in destination"`
	}
}

type RestoreBackupOutput struct {
	Body struct {
		Message    string `json:"message" example:"Database dev_db restored successfully"`
		DurationMs int64  `json:"duration_ms" example:"1250"`
		Logs       string `json:"logs,omitempty"`
	}
}

type DestinationInfo struct {
	Name       string                  `json:"name" example:"offsite_s3"`
	Type       string                  `json:"type" example:"s3"`
	Path       string                  `json:"path" example:"backups/{{ .Database }}.enc"`
	Bucket     string                  `json:"bucket,omitempty" example:"company-backups"`
	Endpoint   string                  `json:"endpoint,omitempty" example:"s3.amazonaws.com"`
	Encrypted  bool                    `json:"encrypted" example:"true"`
	Retention  *config.RetentionPolicy `json:"retention,omitempty"`
}

type DestinationsOutput struct {
	Body []DestinationInfo
}

type HistoryInput struct {
	Limit    int    `query:"limit" default:"20" doc:"Number of records to return"`
	Offset   int    `query:"offset" default:"0" doc:"Offset for pagination"`
	Database string `query:"database" doc:"Filter by database name"`
}

type HistoryOutput struct {
	Body struct {
		Runs  []db.Run `json:"runs"`
		Total int      `json:"total"`
	}
}

type RunDetailInput struct {
	ID string `path:"id" doc:"Run UUID"`
}

type RunDetailOutput struct {
	Body *db.Run
}

type StatsOutput struct {
	Body *db.SummaryStats
}

func (s *Server) registerRoutes(api huma.API) {
	// GET /api/v1/status
	huma.Register(api, huma.Operation{
		OperationID: "get-status",
		Method:      http.MethodGet,
		Path:        "/api/v1/status",
		Summary:     "Get daemon status",
		Tags:        []string{"System"},
	}, func(ctx context.Context, input *struct{}) (*StatusOutput, error) {
		resp := &StatusOutput{}
		resp.Body.Status = "healthy"
		resp.Body.Version = s.version
		resp.Body.UptimeSeconds = int64(time.Since(s.startTime).Seconds())
		resp.Body.DatabaseCount = len(s.cfg.Databases)
		resp.Body.DestinationCount = len(s.cfg.Destinations)
		return resp, nil
	})

	// GET /api/v1/databases
	huma.Register(api, huma.Operation{
		OperationID: "list-databases",
		Method:      http.MethodGet,
		Path:        "/api/v1/databases",
		Summary:     "List configured databases",
		Tags:        []string{"Databases"},
	}, func(ctx context.Context, input *struct{}) (*DatabasesOutput, error) {
		var list []DatabaseInfo
		for name, dbCfg := range s.cfg.Databases {
			info := DatabaseInfo{
				Name:         name,
				Engine:       dbCfg.Engine,
				Host:         dbCfg.Host,
				Port:         dbCfg.Port,
				Database:     dbCfg.Database,
				Schedule:     dbCfg.Schedule,
				Destinations: dbCfg.Destinations,
				IsRunning:    s.runner.IsRunning(name),
				NextRunAt:    s.scheduler.NextRun(name),
			}

			// Get most recent run
			runs, _, err := s.store.GetRecentRuns(ctx, 1, 0, name)
			if err == nil && len(runs) > 0 {
				info.LastRun = &runs[0]
			}

			list = append(list, info)
		}

		sort.Slice(list, func(i, j int) bool {
			return list[i].Name < list[j].Name
		})

		resp := &DatabasesOutput{Body: list}
		return resp, nil
	})

	// POST /api/v1/databases/{id}/backup
	huma.Register(api, huma.Operation{
		OperationID: "trigger-database-backup",
		Method:      http.MethodPost,
		Path:        "/api/v1/databases/{id}/backup",
		Summary:     "Trigger on-demand backup for a database",
		Tags:        []string{"Databases"},
	}, func(ctx context.Context, input *TriggerBackupInput) (*TriggerBackupOutput, error) {
		if _, ok := s.cfg.Databases[input.ID]; !ok {
			return nil, huma.Error404NotFound(fmt.Sprintf("database %q not found", input.ID))
		}

		if s.runner.IsRunning(input.ID) {
			return nil, huma.Error409Conflict(fmt.Sprintf("backup for %q is already running", input.ID))
		}

		// Run asynchronously in background
		go func(name string) {
			log.Info().Str("database", name).Msg("manual backup started via API")
			if _, err := s.runner.RunBackup(context.Background(), name); err != nil {
				log.Error().Err(err).Str("database", name).Msg("manual backup failed")
			}
		}(input.ID)

		resp := &TriggerBackupOutput{}
		resp.Body.Message = fmt.Sprintf("Backup started for database %s", input.ID)
		return resp, nil
	})

	// GET /api/v1/databases/{id}/backups
	huma.Register(api, huma.Operation{
		OperationID: "list-database-backups",
		Method:      http.MethodGet,
		Path:        "/api/v1/databases/{id}/backups",
		Summary:     "List backup files in storage destinations for a database",
		Tags:        []string{"Databases"},
	}, func(ctx context.Context, input *ListBackupsInput) (*ListBackupsOutput, error) {
		if _, ok := s.cfg.Databases[input.ID]; !ok {
			return nil, huma.Error404NotFound(fmt.Sprintf("database %q not found", input.ID))
		}
		items, err := s.runner.ListDatabaseBackups(ctx, input.ID, input.Destination)
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to list database backups", err)
		}
		resp := &ListBackupsOutput{}
		resp.Body.Backups = items
		return resp, nil
	})

	// POST /api/v1/databases/{id}/restore
	huma.Register(api, huma.Operation{
		OperationID: "restore-database-backup",
		Method:      http.MethodPost,
		Path:        "/api/v1/databases/{id}/restore",
		Summary:     "Restore a database from a storage destination backup",
		Tags:        []string{"Databases"},
	}, func(ctx context.Context, input *RestoreBackupInput) (*RestoreBackupOutput, error) {
		if _, ok := s.cfg.Databases[input.ID]; !ok {
			return nil, huma.Error404NotFound(fmt.Sprintf("database %q not found", input.ID))
		}
		if input.Body.Destination == "" || input.Body.BackupPath == "" {
			return nil, huma.Error400BadRequest("destination and backup_path are required")
		}

		var logBuf bytes.Buffer
		startTime := time.Now()
		err := s.runner.RestoreBackup(ctx, input.ID, input.Body.Destination, input.Body.BackupPath, &logBuf)
		durationMs := time.Since(startTime).Milliseconds()

		if err != nil {
			return nil, huma.Error500InternalServerError(fmt.Sprintf("restore failed: %v", err), fmt.Errorf("%s", logBuf.String()))
		}

		resp := &RestoreBackupOutput{}
		resp.Body.Message = fmt.Sprintf("Database %s restored successfully", input.ID)
		resp.Body.DurationMs = durationMs
		resp.Body.Logs = logBuf.String()
		return resp, nil
	})

	// GET /api/v1/destinations
	huma.Register(api, huma.Operation{
		OperationID: "list-destinations",
		Method:      http.MethodGet,
		Path:        "/api/v1/destinations",
		Summary:     "List configured backup destinations",
		Tags:        []string{"Destinations"},
	}, func(ctx context.Context, input *struct{}) (*DestinationsOutput, error) {
		var list []DestinationInfo
		for name, dest := range s.cfg.Destinations {
			info := DestinationInfo{
				Name:      name,
				Type:      string(dest.Type),
				Path:      dest.Path,
				Bucket:    dest.Bucket,
				Endpoint:  dest.Endpoint,
				Encrypted: dest.Encryption != nil && dest.Encryption.Enabled,
				Retention: dest.Retention,
			}
			list = append(list, info)
		}
		sort.Slice(list, func(i, j int) bool {
			return list[i].Name < list[j].Name
		})
		return &DestinationsOutput{Body: list}, nil
	})

	// GET /api/v1/history
	huma.Register(api, huma.Operation{
		OperationID: "get-history",
		Method:      http.MethodGet,
		Path:        "/api/v1/history",
		Summary:     "Get backup run history",
		Tags:        []string{"History"},
	}, func(ctx context.Context, input *HistoryInput) (*HistoryOutput, error) {
		runs, total, err := s.store.GetRecentRuns(ctx, input.Limit, input.Offset, input.Database)
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to query history", err)
		}
		resp := &HistoryOutput{}
		resp.Body.Runs = runs
		resp.Body.Total = total
		return resp, nil
	})

	// GET /api/v1/history/{id}
	huma.Register(api, huma.Operation{
		OperationID: "get-run-detail",
		Method:      http.MethodGet,
		Path:        "/api/v1/history/{id}",
		Summary:     "Get run execution details and logs",
		Tags:        []string{"History"},
	}, func(ctx context.Context, input *RunDetailInput) (*RunDetailOutput, error) {
		run, err := s.store.GetRun(ctx, input.ID)
		if err != nil {
			return nil, huma.Error404NotFound("run not found")
		}
		return &RunDetailOutput{Body: run}, nil
	})

	// GET /api/v1/stats
	huma.Register(api, huma.Operation{
		OperationID: "get-stats",
		Method:      http.MethodGet,
		Path:        "/api/v1/stats",
		Summary:     "Get overall backup statistics",
		Tags:        []string{"System"},
	}, func(ctx context.Context, input *struct{}) (*StatsOutput, error) {
		stats, err := s.store.GetSummaryStats(ctx)
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to get stats", err)
		}
		return &StatsOutput{Body: stats}, nil
	})
}
