// Package cron provides a simple scheduled-job engine for BuckTooth.
// Jobs are configured via CronJobConfig with a time.Duration-style schedule
// (e.g. "5m", "1h") and fire synthetic channel messages into the gateway
// event bus on each tick.
package cron

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/scttfrdmn/bucktooth/internal/channels"
	"github.com/scttfrdmn/bucktooth/internal/config"
)

// FireFunc is called on each job tick with a synthetic inbound message.
type FireFunc func(ctx context.Context, msg *channels.Message)

// JobInfo is a read-only snapshot of a job's configuration, returned by Jobs().
type JobInfo struct {
	Name      string `json:"name"`
	Schedule  string `json:"schedule"`
	ChannelID string `json:"channel_id"`
	UserID    string `json:"user_id"`
	Enabled   bool   `json:"enabled"`
	Message   string `json:"message"`
}

// Scheduler manages a set of periodic jobs.
type Scheduler struct {
	jobs   []*job
	fire   FireFunc
	logger zerolog.Logger
	mu     sync.Mutex
}

type job struct {
	cfg    config.CronJobConfig
	ticker *time.Ticker
	stop   chan struct{}
}

// New creates a Scheduler from config. Only enabled jobs are scheduled.
// Returns an error if any enabled job has an invalid or non-positive schedule.
func New(cfg config.CronConfig, fire FireFunc, logger zerolog.Logger) (*Scheduler, error) {
	s := &Scheduler{
		fire:   fire,
		logger: logger.With().Str("component", "cron").Logger(),
	}
	for _, jcfg := range cfg.Jobs {
		if !jcfg.Enabled {
			continue
		}
		d, err := time.ParseDuration(jcfg.Schedule)
		if err != nil {
			return nil, fmt.Errorf("cron job %q: invalid schedule %q: %w", jcfg.Name, jcfg.Schedule, err)
		}
		if d <= 0 {
			return nil, fmt.Errorf("cron job %q: schedule must be a positive duration", jcfg.Name)
		}
		s.jobs = append(s.jobs, &job{
			cfg:    jcfg,
			ticker: time.NewTicker(d),
			stop:   make(chan struct{}),
		})
	}
	return s, nil
}

// Start launches a goroutine for each scheduled job. Jobs fire until the
// context is cancelled or Stop is called.
func (s *Scheduler) Start(ctx context.Context) {
	for _, j := range s.jobs {
		j := j
		go func() {
			s.logger.Info().
				Str("job", j.cfg.Name).
				Str("schedule", j.cfg.Schedule).
				Msg("cron job started")
			for {
				select {
				case <-j.ticker.C:
					msg := &channels.Message{
						ChannelID: j.cfg.ChannelID,
						UserID:    j.cfg.UserID,
						Content:   j.cfg.Message,
						Timestamp: time.Now(),
					}
					s.logger.Debug().Str("job", j.cfg.Name).Msg("cron job fired")
					s.fire(ctx, msg)
				case <-j.stop:
					s.logger.Debug().Str("job", j.cfg.Name).Msg("cron job stopped")
					return
				case <-ctx.Done():
					return
				}
			}
		}()
	}
}

// Stop stops all job tickers and goroutines. Must be called at most once.
func (s *Scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, j := range s.jobs {
		j.ticker.Stop()
		close(j.stop)
	}
}

// Jobs returns a snapshot of all scheduled jobs.
func (s *Scheduler) Jobs() []JobInfo {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]JobInfo, len(s.jobs))
	for i, j := range s.jobs {
		out[i] = JobInfo{
			Name:      j.cfg.Name,
			Schedule:  j.cfg.Schedule,
			ChannelID: j.cfg.ChannelID,
			UserID:    j.cfg.UserID,
			Enabled:   j.cfg.Enabled,
			Message:   j.cfg.Message,
		}
	}
	return out
}
