package executor_test

import (
	"context"
	"testing"

	"github.com/efekckk/flowgent/internal/executor"
	"github.com/efekckk/flowgent/internal/registry"

	coreloop "github.com/efekckk/flowgent/tools/core.loop"
	coreset "github.com/efekckk/flowgent/tools/core.set"
)

func TestEngine_loopRunsBodyOncePerItem(t *testing.T) {
	reg := registry.New()
	reg.Register("core.loop", coreloop.New())
	reg.Register("core.set", coreset.New())

	bodyRec := &recordingExec{out: map[string]any{"echoed": true}}
	reg.Register("body.echo", bodyRec)

	wf := executor.Workflow{
		Nodes: []executor.Node{
			{ID: "L", Tool: "core.loop", Params: map[string]any{
				"items": []any{"a", "b", "c"},
				"body":  []any{"B"},
			}},
			{ID: "B", Tool: "body.echo", Params: map[string]any{}},
			{ID: "after", Tool: "core.set", Params: map[string]any{
				"values": map[string]any{"final": "{{ $nodes.L.iterations }}"},
			}},
		},
		Edges: []executor.Edge{
			{From: "L", FromPort: "done", To: "after", ToPort: "main"},
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
	if len(bodyRec.calls) != 3 {
		t.Errorf("body calls: %d (want 3)", len(bodyRec.calls))
	}
	afterOut := state.LatestOutput("after")
	if afterOut["final"] != 3 {
		t.Errorf("after.final: %+v", afterOut)
	}
}

func TestEngine_loopWithEmptyItemsFiresOnlyDone(t *testing.T) {
	reg := registry.New()
	reg.Register("core.loop", coreloop.New())
	reg.Register("core.set", coreset.New())
	bodyRec := &recordingExec{out: map[string]any{}}
	reg.Register("body.never", bodyRec)

	wf := executor.Workflow{
		Nodes: []executor.Node{
			{ID: "L", Tool: "core.loop", Params: map[string]any{
				"items": []any{},
				"body":  []any{"B"},
			}},
			{ID: "B", Tool: "body.never", Params: map[string]any{}},
			{ID: "after", Tool: "core.set", Params: map[string]any{
				"values": map[string]any{"done": true},
			}},
		},
		Edges: []executor.Edge{
			{From: "L", FromPort: "done", To: "after", ToPort: "main"},
		},
	}
	eng := executor.NewEngine(reg)
	state, err := eng.Run(context.Background(), &wf, executor.RunOptions{TriggerKind: "manual"})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if len(bodyRec.calls) != 0 {
		t.Errorf("body should not run: %d", len(bodyRec.calls))
	}
	if state.LatestOutput("after")["done"] != true {
		t.Errorf("after did not fire: %+v", state.LatestOutput("after"))
	}
}
