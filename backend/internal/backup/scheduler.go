package backup

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/google/uuid"
)

const minInterval = time.Minute

// Scheduler runs the periodic auto-snapshot job. The interval (in hours,
// possibly fractional) is read from settings at start; Reschedule swaps the
// active job to a new interval without losing the scheduler.
type Scheduler struct {
	svc *Service

	mu        sync.Mutex
	scheduler gocron.Scheduler
	jobID     uuid.UUID
	interval  time.Duration
	started   bool
}

func NewScheduler(svc *Service) *Scheduler {
	return &Scheduler{svc: svc}
}

func (s *Scheduler) Start(intervalHours float64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.started {
		return errors.New("scheduler already started")
	}
	sched, err := gocron.NewScheduler()
	if err != nil {
		return err
	}
	s.scheduler = sched
	s.started = true
	if err := s.scheduleLocked(intervalHours); err != nil {
		_ = s.scheduler.Shutdown()
		s.started = false
		return err
	}
	s.scheduler.Start()
	slog.Info("backup scheduler started", "interval_hours", intervalHours, "interval", s.interval.String())
	return nil
}

func (s *Scheduler) Reschedule(intervalHours float64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.started {
		return errors.New("scheduler not started")
	}
	return s.scheduleLocked(intervalHours)
}

func (s *Scheduler) scheduleLocked(intervalHours float64) error {
	d := time.Duration(intervalHours * float64(time.Hour))
	if d < minInterval {
		d = minInterval
	}
	if s.jobID != uuid.Nil {
		if err := s.scheduler.RemoveJob(s.jobID); err != nil {
			slog.Warn("remove existing backup job", "err", err)
		}
		s.jobID = uuid.Nil
	}
	job, err := s.scheduler.NewJob(
		gocron.DurationJob(d),
		gocron.NewTask(s.runOnce),
		gocron.WithStartAt(gocron.WithStartDateTime(time.Now().Add(d))),
	)
	if err != nil {
		return err
	}
	s.jobID = job.ID()
	s.interval = d
	return nil
}

func (s *Scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.started {
		return
	}
	if err := s.scheduler.Shutdown(); err != nil {
		slog.Warn("scheduler shutdown", "err", err)
	}
	s.started = false
}

func (s *Scheduler) runOnce() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	note := "scheduled"
	n := note
	snap, err := s.svc.Create(ctx, TypeAuto, &n, nil)
	if err != nil {
		slog.Error("scheduled snapshot", "err", err)
		return
	}
	slog.Info("scheduled snapshot created", "id", snap.ID, "filename", snap.Filename, "size_bytes", snap.SizeBytes)
}
