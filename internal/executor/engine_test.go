package executor_test

import (
	"context"
	"errors"
	"testing"

	"github.com/efekckk/flowgent/internal/executor"
	"github.com/efekckk/flowgent/internal/registry"
)

type recordingExec struct {
	calls  []map[string]any
	out    map[string]any
	port   string
	failNx int
	err    error
}

func (r *recordingExec) Execute(_ context.Context, input map[string]any) (registry.ExecuteResult, error) {
	r.calls = append(r.calls, input)
	if r.failNx > 0 {
		r.failNx--
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
	if state.Status != "succeeded" {
		t.Errorf("status: %s", state.Status)
	}
	if state.NodeOutputs["n1"]["hello"] != "world" {
		t.Errorf("output: %+v", state.NodeOutputs["n1"])
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
	if state.Status != "succeeded" {
		t.Errorf("status: %s", state.Status)
	}
	if _, ok := state.NodeOutputs["a"]; !ok {
		t.Errorf("a output missing")
	}
	if _, ok := state.NodeOutputs["b"]; !ok {
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
	if state.NodeStatus["yes"] != "succeeded" {
		t.Errorf("yes status: %s", state.NodeStatus["yes"])
	}
	if state.NodeStatus["no"] != "skipped" {
		t.Errorf("no status (want skipped): %s", state.NodeStatus["no"])
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
	if state.Status != "failed" {
		t.Errorf("status: %s", state.Status)
	}
}
