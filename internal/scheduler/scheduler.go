package scheduler

import (
	"context"
	"fmt"

	"github.com/robfig/cron/v3"
)

type Job func(context.Context) error

type Scheduler struct {
	cron *cron.Cron
}

func New() *Scheduler {
	return &Scheduler{cron: cron.New()}
}

func (s *Scheduler) Register(expr string, job Job) error {
	_, err := s.cron.AddFunc(expr, func() {
		_ = job(context.Background())
	})
	if err != nil {
		return fmt.Errorf("register cron job: %w", err)
	}
	return nil
}

func (s *Scheduler) Start() {
	s.cron.Start()
}

func (s *Scheduler) Stop() {
	stopCtx := s.cron.Stop()
	<-stopCtx.Done()
}
