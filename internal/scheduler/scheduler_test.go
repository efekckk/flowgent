package scheduler_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/efekckk/flowgent/internal/scheduler"
)

type fakeFirer struct {
	calls atomic.Int64
}

func (f *fakeFirer) FireTrigger(ctx context.Context, triggerID, workflowID string, payload map[string]any) error {
	f.calls.Add(1)
	return nil
}

func TestScheduler_RegistersAndFiresOnTick(t *testing.T) {
	firer := &fakeFirer{}
	s := scheduler.New(firer)
	if err := s.Add("trg_a", "wf_a", "@every 1s"); err != nil {
		t.Fatalf("add: %v", err)
	}
	s.Start()
	defer s.Stop()

	deadline := time.After(3 * time.Second)
	for firer.calls.Load() < 1 {
		select {
		case <-deadline:
			t.Fatalf("expected at least 1 fire, got %d", firer.calls.Load())
		default:
			time.Sleep(50 * time.Millisecond)
		}
	}
}

func TestScheduler_InvalidCronRejected(t *testing.T) {
	s := scheduler.New(&fakeFirer{})
	if err := s.Add("trg", "wf", "not-a-cron-expression"); err == nil {
		t.Fatalf("expected error for invalid expression")
	}
}

func TestScheduler_RemoveStopsFiring(t *testing.T) {
	firer := &fakeFirer{}
	s := scheduler.New(firer)
	if err := s.Add("trg", "wf", "@every 1s"); err != nil {
		t.Fatalf("add: %v", err)
	}
	s.Start()
	defer s.Stop()

	time.Sleep(1500 * time.Millisecond)
	before := firer.calls.Load()
	s.Remove("trg")
	time.Sleep(1500 * time.Millisecond)
	after := firer.calls.Load()

	// allow at most 1 in-flight fire after Remove (cron may have already entered the goroutine)
	if after-before > 1 {
		t.Errorf("expected no new fires after Remove, got %d new", after-before)
	}
}

func TestScheduler_LoadFromDB(t *testing.T) {
	firer := &fakeFirer{}
	s := scheduler.New(firer)
	loader := &fakeLoader{rows: []scheduler.LoadedTrigger{
		{ID: "a", WorkflowID: "w1", Expression: "@every 1s"},
		{ID: "b", WorkflowID: "w2", Expression: "@every 1s"},
	}}
	if err := s.LoadFromDB(context.Background(), loader); err != nil {
		t.Fatalf("load: %v", err)
	}
	s.Start()
	defer s.Stop()

	time.Sleep(1500 * time.Millisecond)
	if firer.calls.Load() < 2 {
		t.Errorf("expected >=2 fires (one per trigger), got %d", firer.calls.Load())
	}
}

func TestScheduler_LoadFromDB_SkipsBadExpressions(t *testing.T) {
	firer := &fakeFirer{}
	s := scheduler.New(firer)
	loader := &fakeLoader{rows: []scheduler.LoadedTrigger{
		{ID: "good", WorkflowID: "w1", Expression: "@every 1s"},
		{ID: "bad", WorkflowID: "w2", Expression: "not-a-cron"},
	}}
	if err := s.LoadFromDB(context.Background(), loader); err != nil {
		t.Fatalf("load: %v", err)
	}
	s.Start()
	defer s.Stop()

	time.Sleep(1500 * time.Millisecond)
	if firer.calls.Load() < 1 {
		t.Errorf("good trigger should have fired despite bad sibling; got %d", firer.calls.Load())
	}
}

func TestScheduler_PanicInFirerDoesNotKillCronGoroutine(t *testing.T) {
	firer := &panickyFirer{}
	s := scheduler.New(firer)
	if err := s.Add("trg", "wf", "@every 1s"); err != nil {
		t.Fatalf("add: %v", err)
	}
	s.Start()
	defer s.Stop()

	// Wait for first fire (panic recovered), then confirm a second fire happens.
	time.Sleep(2500 * time.Millisecond)
	if firer.calls.Load() < 2 {
		t.Errorf("expected >=2 fires (panic must not kill the cron goroutine); got %d", firer.calls.Load())
	}
}

type fakeLoader struct{ rows []scheduler.LoadedTrigger }

func (f *fakeLoader) LoadEnabledCronTriggers(ctx context.Context) ([]scheduler.LoadedTrigger, error) {
	return f.rows, nil
}

type panickyFirer struct {
	calls atomic.Int64
}

func (p *panickyFirer) FireTrigger(_ context.Context, _, _ string, _ map[string]any) error {
	p.calls.Add(1)
	panic("simulated firer panic")
}
