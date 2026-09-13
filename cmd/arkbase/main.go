package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/blesswinsamuel/arkbase"
	"github.com/blesswinsamuel/arkbase/internal/api"
	"github.com/blesswinsamuel/arkbase/internal/auth"
	"github.com/blesswinsamuel/arkbase/internal/config"
	"github.com/blesswinsamuel/arkbase/internal/db"
	"github.com/blesswinsamuel/arkbase/internal/engine"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var Version = "1.0.0-dev"

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})

	rootCmd := &cobra.Command{
		Use:     "arkbase",
		Short:   "arkbase is a GitOps-native database backup daemon with an embedded dashboard",
		Version: Version,
	}

	var configFile string

	runCmd := &cobra.Command{
		Use:   "run",
		Short: "Start the arkbase backup scheduler and web dashboard",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDaemon(configFile)
		},
	}
	runCmd.Flags().StringVarP(&configFile, "config", "c", "config.yaml", "Path to config file")

	backupCmd := &cobra.Command{
		Use:   "backup [database]",
		Short: "Execute an immediate backup for a specific database",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runOneOffBackup(configFile, args[0])
		},
	}
	backupCmd.Flags().StringVarP(&configFile, "config", "c", "config.yaml", "Path to config file")

	var destName string
	var backupFile string
	restoreCmd := &cobra.Command{
		Use:   "restore [database]",
		Short: "Restore a database from a backup file in a storage destination",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRestore(configFile, args[0], destName, backupFile)
		},
	}
	restoreCmd.Flags().StringVarP(&configFile, "config", "c", "config.yaml", "Path to config file")
	restoreCmd.Flags().StringVarP(&destName, "destination", "d", "", "Name of storage destination")
	restoreCmd.Flags().StringVarP(&backupFile, "file", "f", "", "Path of backup file in destination")
	_ = restoreCmd.MarkFlagRequired("destination")
	_ = restoreCmd.MarkFlagRequired("file")

	var openAPIOutputFile string
	exportOpenAPICmd := &cobra.Command{
		Use:   "export-openapi",
		Short: "Export OpenAPI 3.1 schema JSON to a file",
		RunE: func(cmd *cobra.Command, args []string) error {
			return exportOpenAPI(openAPIOutputFile)
		},
	}
	exportOpenAPICmd.Flags().StringVarP(&openAPIOutputFile, "out", "o", "openapi.json", "Output file path")

	rootCmd.AddCommand(runCmd, backupCmd, restoreCmd, exportOpenAPICmd)

	// Default to "run" if no subcommand provided
	if len(os.Args) == 1 {
		rootCmd.SetArgs([]string{"run"})
	}

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func runDaemon(configPath string) error {
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return fmt.Errorf("load config %s: %w", configPath, err)
	}

	store, err := db.Open(cfg.Server.DataDir)
	if err != nil {
		return fmt.Errorf("init db: %w", err)
	}
	defer store.Close()

	runner, err := engine.NewRunner(cfg, store)
	if err != nil {
		return fmt.Errorf("init runner: %w", err)
	}

	scheduler, err := engine.NewScheduler(cfg, runner)
	if err != nil {
		return fmt.Errorf("init scheduler: %w", err)
	}

	scheduler.Start()
	defer func() {
		_ = scheduler.Stop()
	}()

	srv := api.NewServer(cfg, store, runner, scheduler, Version)

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	// Apply Basic Auth if enabled
	r.Use(auth.Middleware(cfg.Server.Auth))

	// Mount Huma API routes
	srv.SetupRouter(r)

	// Mount SPA frontend static file server (for any unmatched routes)
	r.NotFound(arkbase.SPAHandler().ServeHTTP)

	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Server.Port),
		Handler: r,
	}

	shutdownChan := make(chan os.Signal, 1)
	signal.Notify(shutdownChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Info().Int("port", cfg.Server.Port).Msg("arkbase server listening")
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("http server error")
		}
	}()

	<-shutdownChan
	log.Info().Msg("shutting down gracefully...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return httpServer.Shutdown(shutdownCtx)
}

func runOneOffBackup(configPath, dbName string) error {
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	store, err := db.Open(cfg.Server.DataDir)
	if err != nil {
		return fmt.Errorf("init db: %w", err)
	}
	defer store.Close()

	runner, err := engine.NewRunner(cfg, store)
	if err != nil {
		return fmt.Errorf("init runner: %w", err)
	}

	run, err := runner.RunBackup(context.Background(), dbName)
	if err != nil {
		return fmt.Errorf("backup failed: %w", err)
	}

	log.Info().Str("run_id", run.ID).Str("status", run.Status).Int64("size", run.SizeBytes).Msg("backup completed")
	return nil
}

func runRestore(configPath, dbName, destName, backupFile string) error {
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	store, err := db.Open(cfg.Server.DataDir)
	if err != nil {
		return fmt.Errorf("init db: %w", err)
	}
	defer store.Close()

	runner, err := engine.NewRunner(cfg, store)
	if err != nil {
		return fmt.Errorf("init runner: %w", err)
	}

	log.Info().Str("database", dbName).Str("destination", destName).Str("file", backupFile).Msg("starting restore...")
	err = runner.RestoreBackup(context.Background(), dbName, destName, backupFile, os.Stdout)
	if err != nil {
		return fmt.Errorf("restore failed: %w", err)
	}

	log.Info().Msg("restore completed successfully")
	return nil
}

func exportOpenAPI(outFile string) error {
	cfg := &config.Config{}
	store, _ := db.Open(os.TempDir())
	runner, _ := engine.NewRunner(cfg, store)
	scheduler, _ := engine.NewScheduler(cfg, runner)
	srv := api.NewServer(cfg, store, runner, scheduler, Version)

	r := chi.NewRouter()
	humaAPI := srv.SetupRouter(r)

	data, err := json.MarshalIndent(humaAPI.OpenAPI(), "", "  ")
	if err != nil {
		return fmt.Errorf("marshal openapi: %w", err)
	}

	if err := os.WriteFile(outFile, data, 0644); err != nil {
		return fmt.Errorf("write %s: %w", outFile, err)
	}

	log.Info().Str("path", outFile).Msg("exported OpenAPI schema")
	return nil
}
