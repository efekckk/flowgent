package executor_test

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/efekckk/flowgent/internal/executor"
	"github.com/efekckk/flowgent/internal/registry"
)

type recordingExec struct {
	mu     sync.Mutex
	calls  []map[string]any
	out    map[string]any
	port   string
	failNx int
	err    error
	slow   time.Duration
}

func (r *recordingExec) Execute(ctx context.Context, input map[string]any) (registry.ExecuteResult, error) {
	r.mu.Lock()
	r.calls = append(r.calls, input)
	failNx := r.failNx
	if r.failNx > 0 {
		r.failNx--
	}
	r.mu.Unlock()
	if r.slow > 0 {
		select {
		case <-time.After(r.slow):
		case <-ctx.Done():
			return registry.ExecuteResult{}, ctx.Err()
		}
	}
	if failNx > 0 {
		return registry.ExecuteResult{}, r.err
	}
	port := r.port
	if port == "" {
		port = "main"
	}
	return registry.ExecuteResult{Output: r.out, Port: port}, nil
}

func TestEngine_singleNode(t *testing.T) {
	reg := registry.New()
	reg.Register("core.set", &recordingExec{out: map[string]any{"hello": "world"}})

	wf := executor.Workflow{
		Nodes: []executor.Node{
			{ID: "n1", Tool: "core.set", Params: map[string]any{"values": map[string]any{"hello": "world"}}},
		},
	}
	eng := executor.NewEngine(reg)
	state, err := eng.Run(context.Background(), &wf, executor.RunOptions{TriggerKind: "manual"})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	runStatus, _ := state.RunStatus()
	if runStatus != "succeeded" {
		t.Errorf("status: %s", runStatus)
	}
	if state.LatestOutput("n1")["hello"] != "world" {
		t.Errorf("output: %+v", state.LatestOutput("n1"))
	}
}

func TestEngine_linearChain(t *testing.T) {
	reg := registry.New()
	reg.Register("core.set", &recordingExec{out: map[string]any{"step": 1}})
	reg.Register("core.wait", &recordingExec{out: map[string]any{"waited_ms": 0}})

	wf := executor.Workflow{
		Nodes: []executor.Node{
			{ID: "a", Tool: "core.set", Params: map[string]any{"values": map[string]any{}}},
			{ID: "b", Tool: "core.wait", Params: map[string]any{"ms": 0}},
		},
		Edges: []executor.Edge{
			{From: "a", FromPort: "main", To: "b", ToPort: "main"},
		},
	}
	eng := executor.NewEngine(reg)
	state, err := eng.Run(context.Background(), &wf, executor.RunOptions{TriggerKind: "manual"})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	runStatus, _ := state.RunStatus()
	if runStatus != "succeeded" {
		t.Errorf("status: %s", runStatus)
	}
	if state.LatestOutput("a") == nil {
		t.Errorf("a output missing")
	}
	if state.LatestOutput("b") == nil {
		t.Errorf("b output missing")
	}
}

func TestEngine_ifBranch_onlyTruePathRuns(t *testing.T) {
	reg := registry.New()
	reg.Register("core.if", &recordingExec{out: map[string]any{"matched": true}, port: "true"})
	tCalls := &recordingExec{out: map[string]any{"true": true}}
	fCalls := &recordingExec{out: map[string]any{"false": true}}
	reg.Register("yes.tool", tCalls)
	reg.Register("no.tool", fCalls)

	wf := executor.Workflow{
		Nodes: []executor.Node{
			{ID: "cond", Tool: "core.if", Params: map[string]any{"left": 1, "op": ">", "right": 0}},
			{ID: "yes", Tool: "yes.tool", Params: map[string]any{}},
			{ID: "no", Tool: "no.tool", Params: map[string]any{}},
		},
		Edges: []executor.Edge{
			{From: "cond", FromPort: "true", To: "yes", ToPort: "main"},
			{From: "cond", FromPort: "false", To: "no", ToPort: "main"},
		},
	}
	eng := executor.NewEngine(reg)
	state, err := eng.Run(context.Background(), &wf, executor.RunOptions{TriggerKind: "manual"})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if state.Status("yes") != "succeeded" {
		t.Errorf("yes status: %s", state.Status("yes"))
	}
	if state.Status("no") != "skipped" {
		t.Errorf("no status (want skipped): %s", state.Status("no"))
	}
}

func TestEngine_nodeFailureMarksRunFailed(t *testing.T) {
	reg := registry.New()
	reg.Register("bad.tool", &recordingExec{failNx: 99, err: errors.New("boom")})

	wf := executor.Workflow{
		Nodes: []executor.Node{
			{ID: "x", Tool: "bad.tool", Params: map[string]any{}},
		},
	}
	eng := executor.NewEngine(reg)
	state, err := eng.Run(context.Background(), &wf, executor.RunOptions{TriggerKind: "manual"})
	if err == nil {
		t.Fatalf("expected error")
	}
	runStatus, _ := state.RunStatus()
	if runStatus != "failed" {
		t.Errorf("status: %s", runStatus)
	}
}

func TestEngine_retryOnRateLimitedThenSucceed(t *testing.T) {
	reg := registry.New()
	rec := &recordingExec{
		out:    map[string]any{"ok": true},
		failNx: 2,
		err:    executor.ErrRateLimited,
	}
	reg.Register("flaky.tool", rec)

	wf := executor.Workflow{
		Nodes: []executor.Node{
			{ID: "n", Tool: "flaky.tool", Params: map[string]any{}},
		},
	}
	eng := executor.NewEngine(reg, executor.WithMaxAttempts(3), executor.WithBackoff(func(int) time.Duration { return 0 }))
	state, err := eng.Run(context.Background(), &wf, executor.RunOptions{TriggerKind: "manual"})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	runStatus, _ := state.RunStatus()
	if runStatus != "succeeded" {
		t.Errorf("status: %s", runStatus)
	}
	if len(rec.calls) != 3 {
		t.Errorf("attempts: %d (want 3)", len(rec.calls))
	}
}

func TestEngine_noRetryOnAuthFailure(t *testing.T) {
	reg := registry.New()
	rec := &recordingExec{failNx: 5, err: executor.ErrAuthFailed}
	reg.Register("bad.tool", rec)

	wf := executor.Workflow{
		Nodes: []executor.Node{
			{ID: "n", Tool: "bad.tool", Params: map[string]any{}},
		},
	}
	eng := executor.NewEngine(reg, executor.WithMaxAttempts(3), executor.WithBackoff(func(int) time.Duration { return 0 }))
	state, err := eng.Run(context.Background(), &wf, executor.RunOptions{TriggerKind: "manual"})
	if err == nil {
		t.Fatalf("expected error")
	}
	runStatus, _ := state.RunStatus()
	if runStatus != "failed" {
		t.Errorf("status: %s", runStatus)
	}
	if len(rec.calls) != 1 {
		t.Errorf("attempts: %d (want 1)", len(rec.calls))
	}
}

func TestEngine_retryExhausted(t *testing.T) {
	reg := registry.New()
	rec := &recordingExec{failNx: 99, err: executor.ErrTransient5xx}
	reg.Register("flaky.tool", rec)

	wf := executor.Workflow{
		Nodes: []executor.Node{
			{ID: "n", Tool: "flaky.tool", Params: map[string]any{}},
		},
	}
	eng := executor.NewEngine(reg, executor.WithMaxAttempts(3), executor.WithBackoff(func(int) time.Duration { return 0 }))
	_, err := eng.Run(context.Background(), &wf, executor.RunOptions{TriggerKind: "manual"})
	if err == nil {
		t.Fatalf("expected error")
	}
	if len(rec.calls) != 3 {
		t.Errorf("attempts: %d (want 3)", len(rec.calls))
	}
}

func TestEngine_parallelBranchesExecuteSimultaneously(t *testing.T) {
	reg := registry.New()
	mk := func() *recordingExec {
		return &recordingExec{out: map[string]any{}, slow: 100 * time.Millisecond}
	}
	reg.Register("slow.a", mk())
	reg.Register("slow.b", mk())
	reg.Register("slow.c", mk())
	reg.Register("core.set", &recordingExec{out: map[string]any{}})

	wf := executor.Workflow{
		Nodes: []executor.Node{
			{ID: "fan", Tool: "core.set", Params: map[string]any{"values": map[string]any{}}},
			{ID: "a", Tool: "slow.a", Params: map[string]any{}},
			{ID: "b", Tool: "slow.b", Params: map[string]any{}},
			{ID: "c", Tool: "slow.c", Params: map[string]any{}},
		},
		Edges: []executor.Edge{
			{From: "fan", FromPort: "main", To: "a", ToPort: "main"},
			{From: "fan", FromPort: "main", To: "b", ToPort: "main"},
			{From: "fan", FromPort: "main", To: "c", ToPort: "main"},
		},
	}
	eng := executor.NewEngine(reg, executor.WithMaxParallel(8))
	start := time.Now()
	state, err := eng.Run(context.Background(), &wf, executor.RunOptions{TriggerKind: "manual"})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if got := time.Since(start); got > 200*time.Millisecond {
		t.Errorf("parallel execution took too long: %v (expected ~100ms)", got)
	}
	runStatus, _ := state.RunStatus()
	if runStatus != "succeeded" {
		t.Errorf("status: %s", runStatus)
	}
}

func TestEngine_errorPortRoutesToCleanup(t *testing.T) {
	reg := registry.New()
	reg.Register("bad.tool", &recordingExec{failNx: 99, err: executor.ErrAuthFailed})
	reg.Register("cleanup.tool", &recordingExec{out: map[string]any{"cleaned": true}})

	wf := executor.Workflow{
		Nodes: []executor.Node{
			{ID: "fail", Tool: "bad.tool", Params: map[string]any{}},
			{ID: "cleanup", Tool: "cleanup.tool", Params: map[string]any{}},
		},
		Edges: []executor.Edge{
			{From: "fail", FromPort: "error", To: "cleanup", ToPort: "main"},
		},
	}
	eng := executor.NewEngine(reg, executor.WithMaxAttempts(1))
	state, err := eng.Run(context.Background(), &wf, executor.RunOptions{TriggerKind: "manual"})
	if err != nil {
		t.Fatalf("run should not fail when error edge is present: %v", err)
	}
	runStatus, _ := state.RunStatus()
	if runStatus != "succeeded" {
		t.Errorf("status: %s", runStatus)
	}
	if state.Status("cleanup") != "succeeded" {
		t.Errorf("cleanup status: %s", state.Status("cleanup"))
	}
}

func TestEngine_noErrorEdgeStillFailsRun(t *testing.T) {
	reg := registry.New()
	reg.Register("bad.tool", &recordingExec{failNx: 99, err: executor.ErrAuthFailed})

	wf := executor.Workflow{
		Nodes: []executor.Node{
			{ID: "fail", Tool: "bad.tool", Params: map[string]any{}},
		},
	}
	eng := executor.NewEngine(reg, executor.WithMaxAttempts(1))
	state, err := eng.Run(context.Background(), &wf, executor.RunOptions{TriggerKind: "manual"})
	if err == nil {
		t.Fatalf("expected error")
	}
	runStatus, _ := state.RunStatus()
	if runStatus != "failed" {
		t.Errorf("status: %s", runStatus)
	}
}

// fakeRunStore is a minimal in-memory RunStore used by the RunFromReplay
// test. It captures what the engine would have written so the test can
// assert on the new run's wiring (parent link, payload clone) without
// touching Postgres.
type fakeRunStore struct {
	wfDef       *executor.Workflow
	wfVersion   int
	parentRunID string
	parentPay   []byte

	inserts []executor.InsertRunParams
	persist struct {
		runID   string
		status  string
		errMsg  string
		called  bool
		started time.Time
		ended   time.Time
	}
}

func (f *fakeRunStore) LoadWorkflowForRun(_ context.Context, _ string) (int, *executor.Workflow, error) {
	return f.wfVersion, f.wfDef, nil
}

func (f *fakeRunStore) InsertRun(_ context.Context, p executor.InsertRunParams) error {
	f.inserts = append(f.inserts, p)
	return nil
}

func (f *fakeRunStore) PersistRun(_ context.Context, runID string, _ *executor.Workflow, _ *executor.RunState,
	status, errMsg string, started, ended time.Time) error {
	f.persist.called = true
	f.persist.runID = runID
	f.persist.status = status
	f.persist.errMsg = errMsg
	f.persist.started = started
	f.persist.ended = ended
	return nil
}

func (f *fakeRunStore) GetTriggerPayload(_ context.Context, runID string) (json.RawMessage, error) {
	if runID != f.parentRunID {
		return nil, errors.New("unknown parent")
	}
	return json.RawMessage(f.parentPay), nil
}

func TestEngine_RunFromReplay_linksParentAndClonesPayload(t *testing.T) {
	reg := registry.New()
	reg.Register("core.set", &recordingExec{out: map[string]any{"ok": true}})

	wf := &executor.Workflow{
		Nodes: []executor.Node{
			{ID: "n1", Tool: "core.set", Params: map[string]any{"values": map[string]any{}}},
		},
	}
	store := &fakeRunStore{
		wfDef:       wf,
		wfVersion:   3,
		parentRunID: "run_parent",
		parentPay:   []byte(`{"hello":"world"}`),
	}

	eng := executor.NewEngine(reg,
		executor.WithRunStore(store),
		executor.WithRunIDGenerator(func() string { return "run_replay_1" }),
	)

	newID, err := eng.RunFromReplay(context.Background(), "wf_abc", "run_parent",
		map[string]any{"hello": "world"})
	if err != nil {
		t.Fatalf("replay: %v", err)
	}
	if newID != "run_replay_1" {
		t.Fatalf("new run id: %q", newID)
	}

	if len(store.inserts) != 1 {
		t.Fatalf("expected 1 insert, got %d", len(store.inserts))
	}
	got := store.inserts[0]
	if got.ID != "run_replay_1" {
		t.Errorf("inserted id: %q", got.ID)
	}
	if got.WorkflowID != "wf_abc" {
		t.Errorf("workflow id: %q", got.WorkflowID)
	}
	if got.WorkflowVersion != 3 {
		t.Errorf("version: %d", got.WorkflowVersion)
	}
	if got.TriggerKind != "replay" {
		t.Errorf("trigger kind: %q", got.TriggerKind)
	}
	if got.ParentRunID == nil || *got.ParentRunID != "run_parent" {
		t.Errorf("parent run id: %v", got.ParentRunID)
	}
	if string(got.TriggerPayload) != `{"hello":"world"}` {
		t.Errorf("payload: %s", string(got.TriggerPayload))
	}
	if !store.persist.called || store.persist.status != "succeeded" {
		t.Errorf("persist not called or wrong status: %+v", store.persist)
	}
}

func TestEngine_RunFromReplay_requiresRunStore(t *testing.T) {
	reg := registry.New()
	eng := executor.NewEngine(reg)
	if _, err := eng.RunFromReplay(context.Background(), "wf", "run", nil); !errors.Is(err, executor.ErrNoRunStore) {
		t.Fatalf("expected ErrNoRunStore, got %v", err)
	}
}
