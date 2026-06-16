package scheduler

import (
	"context"
	"fmt"
	"sync"

	"github.com/robfig/cron/v3"
)

// Firer dispatches a trigger fire into the workflow engine. Implementations
// typically resolve the latest workflow version, create a workflow_run row,
// stream node events to run_logs, and return when the run has at least been
// scheduled (not necessarily completed).
type Firer interface {
	FireTrigger(ctx context.Context, triggerID, workflowID string, payload map[string]any) error
}

// LoadedTrigger is the minimal trigger info the scheduler needs at boot.
type LoadedTrigger struct {
	ID         string
	WorkflowID string
	Expression string
}

// Loader hands the scheduler the set of enabled cron triggers to register.
// Implementations typically wrap storage.TriggerRepo.ListEnabledByKind("cron").
type Loader interface {
	LoadEnabledCronTriggers(ctx context.Context) ([]LoadedTrigger, error)
}

// Scheduler wraps robfig/cron with the Flowgent trigger boot lifecycle. On
// server start, LoadFromDB pulls every enabled cron trigger from the
// triggers table and registers it with the underlying cron engine. Each
// fire calls the injected Firer, which is responsible for creating a
// workflow run record and dispatching it to the engine. The scheduler does
// NOT own the run record; it only times the dispatch.
type Scheduler struct {
	firer Firer
	cron  *cron.Cron
	mu    sync.Mutex
	jobs  map[string]cron.EntryID
}

func New(firer Firer) *Scheduler {
	return &Scheduler{
		firer: firer,
		cron: cron.New(cron.WithParser(cron.NewParser(
			cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor,
		))),
		jobs: make(map[string]cron.EntryID),
	}
}

// Add registers (or replaces) a cron expression for triggerID. Replacing an
// existing entry is intentional: when the API handler updates a trigger's
// cron expression it re-registers without restarting the whole engine.
func (s *Scheduler) Add(triggerID, workflowID, expression string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if prev, ok := s.jobs[triggerID]; ok {
		s.cron.Remove(prev)
		delete(s.jobs, triggerID)
	}
	entryID, err := s.cron.AddFunc(expression, func() {
		// Recover firer panics so a buggy tool can never kill the cron goroutine.
		defer func() { _ = recover() }()
		_ = s.firer.FireTrigger(context.Background(), triggerID, workflowID, map[string]any{
			"cron": expression,
		})
	})
	if err != nil {
		return fmt.Errorf("scheduler: invalid expression %q: %w", expression, err)
	}
	s.jobs[triggerID] = entryID
	return nil
}

func (s *Scheduler) Remove(triggerID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if id, ok := s.jobs[triggerID]; ok {
		s.cron.Remove(id)
		delete(s.jobs, triggerID)
	}
}

// Start the cron loop. Returns immediately; ticks run on a background goroutine.
func (s *Scheduler) Start() { s.cron.Start() }

// Stop blocks until the currently-firing job (if any) finishes, then halts.
func (s *Scheduler) Stop() {
	ctx := s.cron.Stop()
	<-ctx.Done()
}

// LoadFromDB pulls every enabled cron trigger and registers each one.
// Individual bad expressions are skipped so a single misconfiguration cannot
// block boot.
func (s *Scheduler) LoadFromDB(ctx context.Context, l Loader) error {
	rows, err := l.LoadEnabledCronTriggers(ctx)
	if err != nil {
		return fmt.Errorf("scheduler: load: %w", err)
	}
	for _, r := range rows {
		if err := s.Add(r.ID, r.WorkflowID, r.Expression); err != nil {
			// Bad expression — skip. The trigger remains in DB; the API
			// handler will reject the same value on next edit attempt.
			continue
		}
	}
	return nil
}
