package executor_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/efekckk/flowgent/internal/executor"
	"github.com/efekckk/flowgent/internal/registry"

	coreif "github.com/efekckk/flowgent/tools/core.if"
	coreset "github.com/efekckk/flowgent/tools/core.set"
	corewait "github.com/efekckk/flowgent/tools/core.wait"
	httprequest "github.com/efekckk/flowgent/tools/http.request"
)

func TestEngine_endToEnd_setWaitIfHttp(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"echoed": true}`))
	}))
	defer srv.Close()

	reg := registry.New()
	reg.Register("core.set", coreset.New())
	reg.Register("core.wait", corewait.New())
	reg.Register("core.if", coreif.New())
	reg.Register("http.request", httprequest.New())

	wf := executor.Workflow{
		Nodes: []executor.Node{
			{
				ID:     "seed",
				Tool:   "core.set",
				Params: map[string]any{"values": map[string]any{"amount": 150}},
			},
			{
				ID:     "pause",
				Tool:   "core.wait",
				Params: map[string]any{"ms": 1},
			},
			{
				ID:   "decide",
				Tool: "core.if",
				Params: map[string]any{
					"left":  "{{ $nodes.seed.amount }}",
					"op":    ">",
					"right": 100,
				},
			},
			{
				ID:   "call",
				Tool: "http.request",
				Params: map[string]any{
					"method": "GET",
					"url":    srv.URL,
				},
			},
			{
				ID:     "skipped",
				Tool:   "core.set",
				Params: map[string]any{"values": map[string]any{"why": "low"}},
			},
		},
		Edges: []executor.Edge{
			{From: "seed", FromPort: "main", To: "pause", ToPort: "main"},
			{From: "pause", FromPort: "main", To: "decide", ToPort: "main"},
			{From: "decide", FromPort: "true", To: "call", ToPort: "main"},
			{From: "decide", FromPort: "false", To: "skipped", ToPort: "main"},
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
	if state.NodeStatus["call"] != "succeeded" {
		t.Errorf("call should have run: %+v", state.NodeStatus)
	}
	if state.NodeStatus["skipped"] != "skipped" {
		t.Errorf("skipped should be skipped: %+v", state.NodeStatus)
	}
	if state.NodeOutputs["call"]["status"] != 200 {
		t.Errorf("http status: %+v", state.NodeOutputs["call"])
	}
}
