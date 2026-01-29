package scheduler

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/murr/check-and-ping/internal/check"
	"github.com/murr/check-and-ping/internal/claude"
	"github.com/murr/check-and-ping/internal/notifier"
	"github.com/murr/check-and-ping/internal/state"
)

const (
	maxBackoffMultiplier = 32  // Max 32x the base interval
	maxBackoffDuration   = time.Hour
)

// Scheduler runs checks at configured intervals
type Scheduler struct {
	checks   []check.Check
	claude   *claude.Client
	notifier notifier.Notifier
	state    state.State
	logger   *log.Logger

	wg     sync.WaitGroup
	cancel context.CancelFunc
}

// New creates a new scheduler
func New(claude *claude.Client, notifier notifier.Notifier, state state.State, logger *log.Logger) *Scheduler {
	if logger == nil {
		logger = log.Default()
	}

	return &Scheduler{
		claude:   claude,
		notifier: notifier,
		state:    state,
		logger:   logger,
	}
}

// Register adds a check to the scheduler
func (s *Scheduler) Register(c check.Check) {
	s.checks = append(s.checks, c)
}

// Start begins running all registered checks
func (s *Scheduler) Start(ctx context.Context) {
	ctx, s.cancel = context.WithCancel(ctx)

	for _, c := range s.checks {
		s.wg.Add(1)
		go s.runCheck(ctx, c)
	}
}

// Stop gracefully shuts down the scheduler
func (s *Scheduler) Stop() {
	if s.cancel != nil {
		s.cancel()
	}
	s.wg.Wait()
}

// runCheck runs a single check on its interval with exponential backoff
func (s *Scheduler) runCheck(ctx context.Context, c check.Check) {
	defer s.wg.Done()

	backoffMultiplier := 1
	consecutiveFailures := 0

	// Run immediately on start
	s.executeCheck(ctx, c, &backoffMultiplier, &consecutiveFailures)

	for {
		// Calculate next interval with backoff
		interval := c.Interval * time.Duration(backoffMultiplier)
		if interval > maxBackoffDuration {
			interval = maxBackoffDuration
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(interval):
			s.executeCheck(ctx, c, &backoffMultiplier, &consecutiveFailures)
		}
	}
}

func (s *Scheduler) executeCheck(ctx context.Context, c check.Check, backoffMultiplier *int, consecutiveFailures *int) {
	s.logger.Printf("[%s] running check", c.Name)

	result, err := c.Run(ctx, s.claude)
	if err != nil {
		*consecutiveFailures++
		*backoffMultiplier = min(1<<*consecutiveFailures, maxBackoffMultiplier)
		s.logger.Printf("[%s] check error (backoff %dx): %v", c.Name, *backoffMultiplier, err)
		return
	}

	// Reset backoff on success
	*consecutiveFailures = 0
	*backoffMultiplier = 1

	if !result.ShouldAlert {
		s.logger.Printf("[%s] no alert needed", c.Name)
		// Clear state when condition clears
		if err := s.state.Clear(c.Name); err != nil {
			s.logger.Printf("[%s] failed to clear state: %v", c.Name, err)
		}
		return
	}

	// Check if we should send this alert (avoid duplicates)
	resultHash := state.Hash(result.Title, result.Message)
	if !s.state.ShouldAlert(c.Name, resultHash) {
		s.logger.Printf("[%s] duplicate alert suppressed", c.Name)
		return
	}

	// Send alert
	alert := check.NewAlertFromResult(c.Name, result)
	if err := s.notifier.Send(ctx, alert); err != nil {
		s.logger.Printf("[%s] notification error: %v", c.Name, err)
		return
	}

	// Mark as alerted
	if err := s.state.MarkAlerted(c.Name, resultHash); err != nil {
		s.logger.Printf("[%s] failed to mark alerted: %v", c.Name, err)
	}

	s.logger.Printf("[%s] alert sent: %s", c.Name, result.Title)
}
