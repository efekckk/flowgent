package executor_test

import (
	"context"
	"testing"

	"github.com/efekckk/flowgent/internal/executor"
	"github.com/efekckk/flowgent/internal/registry"

	coremerge "github.com/efekckk/flowgent/tools/core.merge"
)

func TestEngine_mergeWaitsForBothUpstream(t *testing.T) {
	reg := registry.New()
	reg.Register("core.set", &recordingExec{out: map[string]any{}})
	reg.Register("upstream.a", &recordingExec{out: map[string]any{"k": "a"}})
	reg.Register("upstream.b", &recordingExec{out: map[string]any{"k": "b"}})
	reg.Register("core.merge", coremerge.New())

	wf := executor.Workflow{
		Nodes: []executor.Node{
			{ID: "fan", Tool: "core.set", Params: map[string]any{"values": map[string]any{}}},
			{ID: "a", Tool: "upstream.a", Params: map[string]any{}},
			{ID: "b", Tool: "upstream.b", Params: map[string]any{}},
			{ID: "m", Tool: "core.merge", Params: map[string]any{
				"mode":   "append",
				"inputs": "$merge.upstream",
			}},
		},
		Edges: []executor.Edge{
			{From: "fan", FromPort: "main", To: "a", ToPort: "main"},
			{From: "fan", FromPort: "main", To: "b", ToPort: "main"},
			{From: "a", FromPort: "main", To: "m", ToPort: "main"},
			{From: "b", FromPort: "main", To: "m", ToPort: "main"},
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
	out := state.LatestOutput("m")
	items, _ := out["items"].([]any)
	if len(items) != 2 {
		t.Errorf("items: %+v", items)
	}
}
