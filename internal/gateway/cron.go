package gateway

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

// CronJob is a scheduled action: at the configured cadence, RunAction is
// executed. RunAction is opaque to the scheduler — it is just a verb the
// REST handler dispatches against the wa manager (e.g. "send_message").
type CronJob struct {
	ID         string            `yaml:"id"          json:"id"`
	Name       string            `yaml:"name"        json:"name"`
	Cron       string            `yaml:"cron"        json:"cron"`        // "@every 5m" or "@daily 09:00"
	RunAction  string            `yaml:"action"      json:"action"`      // "send_message" | "mark_read" | "set_presence"
	RunArgs    map[string]string `yaml:"args"        json:"args"`
	Enabled    bool              `yaml:"enabled"     json:"enabled"`
	LastRun    time.Time         `yaml:"last_run"    json:"last_run"`
	LastResult string            `yaml:"last_result" json:"last_result"` // "ok" / "error: ..."
	CreatedAt  time.Time         `yaml:"created_at"  json:"created_at"`
}

// ActionFunc is the contract the gateway uses to actually execute a job.
// Implemented in app.go against the wa manager.
type ActionFunc func(action string, args map[string]string) error

// Scheduler runs CronJobs. It evaluates schedules every Tick interval and
// invokes the registered ActionFunc for due jobs.
type Scheduler struct {
	mu       sync.Mutex
	jobs     []CronJob
	act      ActionFunc
	stopCh   chan struct{}
	tick     time.Duration
	lastEval map[string]time.Time // jobID -> last evaluated "next fire" to avoid double-firing
}

// NewScheduler creates a Scheduler that ticks every interval. action is
// the dispatcher; it must be safe for concurrent use.
func NewScheduler(interval time.Duration, action ActionFunc) *Scheduler {
	return &Scheduler{
		act:      action,
		stopCh:   make(chan struct{}),
		tick:     interval,
		lastEval: map[string]time.Time{},
	}
}

// Start launches the ticker. Stop with the returned cancel func.
func (s *Scheduler) Start() {
	go s.loop()
}

func (s *Scheduler) loop() {
	t := time.NewTicker(s.tick)
	defer t.Stop()
	for {
		select {
		case <-s.stopCh:
			return
		case now := <-t.C:
			s.evaluate(now)
		}
	}
}

// Stop signals the loop to exit.
func (s *Scheduler) Stop() { close(s.stopCh) }

func (s *Scheduler) evaluate(now time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.jobs {
		j := &s.jobs[i]
		if !j.Enabled {
			continue
		}
		next, ok := nextFire(j.Cron, now)
		if !ok {
			continue
		}
		last, seen := s.lastEval[j.ID]
		if seen && !last.Before(next) {
			continue // already evaluated this fire time
		}
		s.lastEval[j.ID] = next
		go s.run(j, next)
	}
}

func (s *Scheduler) run(j *CronJob, when time.Time) {
	err := s.act(j.RunAction, j.RunArgs)
	s.mu.Lock()
	j.LastRun = when
	if err != nil {
		j.LastResult = "error: " + err.Error()
	} else {
		j.LastResult = "ok"
	}
	s.mu.Unlock()
}

// Add appends a job.
func (s *Scheduler) Add(j CronJob) (CronJob, error) {
	if j.Cron == "" {
		return CronJob{}, fmt.Errorf("cron is required")
	}
	if j.RunAction == "" {
		return CronJob{}, fmt.Errorf("action is required")
	}
	if _, ok := nextFire(j.Cron, time.Now()); !ok {
		return CronJob{}, fmt.Errorf("invalid cron expression: %q", j.Cron)
	}
	s.mu.Lock()
	if j.ID == "" {
		j.ID = newCronID()
	}
	if j.CreatedAt.IsZero() {
		j.CreatedAt = time.Now()
	}
	j.Enabled = true
	s.jobs = append(s.jobs, j)
	s.mu.Unlock()
	return j, nil
}

// List returns a copy of current jobs.
func (s *Scheduler) List() []CronJob {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]CronJob, len(s.jobs))
	copy(out, s.jobs)
	return out
}

// Remove deletes a job by ID.
func (s *Scheduler) Remove(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, j := range s.jobs {
		if j.ID == id {
			s.jobs = append(s.jobs[:i], s.jobs[i+1:]...)
			delete(s.lastEval, id)
			return nil
		}
	}
	return fmt.Errorf("job %q not found", id)
}

func newCronID() string {
	var b [6]byte
	_, _ = rand.Read(b[:])
	return "cron_" + hex.EncodeToString(b[:])
}
