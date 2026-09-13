package engine

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/blesswinsamuel/arkbase/internal/config"
	"github.com/go-co-op/gocron/v2"
	"github.com/rs/zerolog/log"
)

type Scheduler struct {
	cfg     *config.Config
	runner  *Runner
	cron    gocron.Scheduler
	jobsMu  sync.RWMutex
	jobMap  map[string]gocron.Job
}

func NewScheduler(cfg *config.Config, runner *Runner) (*Scheduler, error) {
	s, err := gocron.NewScheduler()
	if err != nil {
		return nil, fmt.Errorf("init gocron scheduler: %w", err)
	}

	sched := &Scheduler{
		cfg:    cfg,
		runner: runner,
		cron:   s,
		jobMap: make(map[string]gocron.Job),
	}

	for dbName, dbCfg := range cfg.Databases {
		if dbCfg.Schedule == "" {
			continue
		}

		name := dbName
		job, err := s.NewJob(
			gocron.CronJob(dbCfg.Schedule, false),
			gocron.NewTask(func() {
				log.Info().Str("database", name).Msg("scheduled backup job triggered")
				_, runErr := runner.RunBackup(context.Background(), name)
				if runErr != nil {
					log.Error().Err(runErr).Str("database", name).Msg("scheduled backup job failed")
				}
			}),
			gocron.WithName(name),
		)
		if err != nil {
			log.Error().Err(err).Str("database", name).Str("schedule", dbCfg.Schedule).Msg("failed to register cron job")
			continue
		}

		sched.jobMap[name] = job
		log.Info().Str("database", name).Str("schedule", dbCfg.Schedule).Msg("registered backup schedule")
	}

	return sched, nil
}

func (s *Scheduler) Start() {
	s.cron.Start()
}

func (s *Scheduler) Stop() error {
	return s.cron.Shutdown()
}

func (s *Scheduler) NextRun(dbName string) *time.Time {
	s.jobsMu.RLock()
	defer s.jobsMu.RUnlock()

	job, ok := s.jobMap[dbName]
	if !ok {
		return nil
	}

	next, err := job.NextRun()
	if err != nil {
		return nil
	}
	return &next
}
