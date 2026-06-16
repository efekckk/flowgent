package executor_test

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/efekckk/flowgent/internal/executor"
	"github.com/efekckk/flowgent/internal/registry"
)

type captureSink struct {
	mu   sync.Mutex
	rows []captureRow
}

type captureRow struct {
	RunID, NodeID, Level, Message string
}

func (c *captureSink) Append(_ context.Context, runID, nodeID, level, msg string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.rows = append(c.rows, captureRow{RunID: runID, NodeID: nodeID, Level: level, Message: msg})
}

func (c *captureSink) snapshot() []captureRow {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]captureRow, len(c.rows))
	copy(out, c.rows)
	return out
}

// leakingCredentialResolver returns the provided secret verbatim so the
// engine has something to inject as "__credential". The credential
// regression test asserts none of this leaks into log messages.
type leakingCredentialResolver struct {
	secret string
}

func (r *leakingCredentialResolver) Resolve(_ context.Context, ref string) (map[string]any, error) {
	return map[string]any{"id": ref, "password": r.secret}, nil
}

func TestEngine_LogSink_EmitsStartedAndSucceededPerNode(t *testing.T) {
	reg := registry.New()
	reg.Register("core.set", &recordingExec{out: map[string]any{"k": "v"}})
	reg.Register("core.wait", &recordingExec{out: map[string]any{"waited_ms": 0}})

	wf := executor.Workflow{
		Nodes: []executor.Node{
			{ID: "a", Tool: "core.set", Params: map[string]any{}},
			{ID: "b", Tool: "core.wait", Params: map[string]any{"ms": 0}},
		},
		Edges: []executor.Edge{{From: "a", FromPort: "main", To: "b", ToPort: "main"}},
	}
	sink := &captureSink{}
	eng := executor.NewEngine(reg, executor.WithLogSink(sink))
	if _, err := eng.Run(context.Background(), &wf, executor.RunOptions{TriggerKind: "manual", RunID: "run_42"}); err != nil {
		t.Fatalf("run: %v", err)
	}

	rows := sink.snapshot()
	started, succeeded := 0, 0
	for _, r := range rows {
		if r.RunID != "run_42" {
			t.Errorf("run id not propagated: %+v", r)
		}
		if strings.Contains(r.Message, ": started") {
			started++
		}
		if strings.Contains(r.Message, ": succeeded") {
			succeeded++
		}
	}
	if started < 2 || succeeded < 2 {
		t.Errorf("expected >=2 started + >=2 succeeded; got started=%d succeeded=%d rows=%+v",
			started, succeeded, rows)
	}
}

func TestEngine_LogSink_NeverLeaksCredential(t *testing.T) {
	const secret = "leak-me-pw-xyz"
	resolver := &leakingCredentialResolver{secret: secret}
	reg := registry.New()
	reg.Register("needs.credential", &recordingExec{out: map[string]any{"ok": true}})

	wf := executor.Workflow{
		Nodes: []executor.Node{
			{ID: "n", Tool: "needs.credential", Credential: "cred_xyz", Params: map[string]any{}},
		},
	}
	sink := &captureSink{}
	eng := executor.NewEngine(reg,
		executor.WithCredentialResolver(resolver),
		executor.WithLogSink(sink),
	)
	if _, err := eng.Run(context.Background(), &wf, executor.RunOptions{TriggerKind: "manual", RunID: "run_cred"}); err != nil {
		t.Fatalf("run: %v", err)
	}

	rows := sink.snapshot()
	if len(rows) == 0 {
		t.Fatalf("expected some log rows for the run")
	}
	for _, r := range rows {
		if strings.Contains(r.Message, secret) {
			t.Errorf("credential value leaked into log row: %+v", r)
		}
		if strings.Contains(r.Message, "__credential") {
			t.Errorf("__credential key leaked into log row: %+v", r)
		}
	}
}

func TestEngine_LogSink_NoopWhenNotConfigured(t *testing.T) {
	reg := registry.New()
	reg.Register("core.set", &recordingExec{out: map[string]any{"ok": true}})

	wf := executor.Workflow{
		Nodes: []executor.Node{
			{ID: "n", Tool: "core.set", Params: map[string]any{}},
		},
	}
	eng := executor.NewEngine(reg) // no WithLogSink
	state, err := eng.Run(context.Background(), &wf, executor.RunOptions{TriggerKind: "manual"})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if status, _ := state.RunStatus(); status != "succeeded" {
		t.Errorf("status: %s", status)
	}

	// Explicit nil sink should also be a no-op (not a panic).
	eng2 := executor.NewEngine(reg, executor.WithLogSink(nil))
	if _, err := eng2.Run(context.Background(), &wf, executor.RunOptions{TriggerKind: "manual"}); err != nil {
		t.Fatalf("run with nil sink: %v", err)
	}
}

func TestEngine_LogSink_FailedNodeEmitsError(t *testing.T) {
	reg := registry.New()
	reg.Register("bad.tool", &recordingExec{failNx: 99, err: errors.New("boom")})

	wf := executor.Workflow{
		Nodes: []executor.Node{
			{ID: "fetch", Tool: "bad.tool", Params: map[string]any{}},
		},
	}
	sink := &captureSink{}
	eng := executor.NewEngine(reg,
		executor.WithMaxAttempts(1),
		executor.WithLogSink(sink),
	)
	if _, err := eng.Run(context.Background(), &wf, executor.RunOptions{TriggerKind: "manual"}); err == nil {
		t.Fatalf("expected run error")
	}

	gotError := false
	for _, r := range sink.snapshot() {
		if r.Level == "error" && r.NodeID == "fetch" {
			gotError = true
		}
	}
	if !gotError {
		t.Errorf("expected error-level row for failed node; rows=%+v", sink.snapshot())
	}
}
