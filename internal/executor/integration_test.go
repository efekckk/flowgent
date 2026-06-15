package executor_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/efekckk/flowgent/internal/executor"
	"github.com/efekckk/flowgent/internal/registry"

	corecode "github.com/efekckk/flowgent/tools/core.code"
	coreif "github.com/efekckk/flowgent/tools/core.if"
	coreloop "github.com/efekckk/flowgent/tools/core.loop"
	coremerge "github.com/efekckk/flowgent/tools/core.merge"
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
	runStatus, _ := state.RunStatus()
	if runStatus != "succeeded" {
		t.Errorf("status: %s", runStatus)
	}
	if state.Status("call") != "succeeded" {
		t.Errorf("call should have run: %+v", state.Statuses())
	}
	if state.Status("skipped") != "skipped" {
		t.Errorf("skipped should be skipped: %+v", state.Statuses())
	}
	if state.LatestOutput("call")["status"] != 200 {
		t.Errorf("http status: %+v", state.LatestOutput("call"))
	}
}

func TestEngine_endToEnd_parallelMergeLoopCode(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"upstream": "ok"}`))
	}))
	defer srv.Close()

	reg := registry.New()
	reg.Register("core.set", coreset.New())
	reg.Register("core.wait", corewait.New())
	reg.Register("core.if", coreif.New())
	reg.Register("http.request", httprequest.New())
	reg.Register("core.merge", coremerge.New())
	reg.Register("core.loop", coreloop.New())
	reg.Register("core.code", corecode.New())

	wf := executor.Workflow{
		Nodes: []executor.Node{
			// Two parallel HTTP fetches feed into a merge
			{ID: "fetch_a", Tool: "http.request", Params: map[string]any{"method": "GET", "url": srv.URL}},
			{ID: "fetch_b", Tool: "http.request", Params: map[string]any{"method": "GET", "url": srv.URL}},
			{ID: "merged", Tool: "core.merge", Params: map[string]any{"mode": "append", "inputs": "$merge.upstream"}},
			// Loop over merged.items
			{ID: "L", Tool: "core.loop", Params: map[string]any{
				"items": "{{ $nodes.merged.items }}",
				"body":  []any{"transform"},
			}},
			{ID: "transform", Tool: "core.code", Params: map[string]any{
				"code": `return { iteration: $now };`,
			}},
			{ID: "done", Tool: "core.set", Params: map[string]any{
				"values": map[string]any{"final": "{{ $nodes.L.iterations }}"},
			}},
		},
		Edges: []executor.Edge{
			{From: "fetch_a", FromPort: "main", To: "merged", ToPort: "main"},
			{From: "fetch_b", FromPort: "main", To: "merged", ToPort: "main"},
			{From: "merged", FromPort: "main", To: "L", ToPort: "main"},
			{From: "L", FromPort: "done", To: "done", ToPort: "main"},
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
	v := state.LatestOutput("done")["final"]
	if v != 2 && v != int64(2) && v != float64(2) {
		t.Errorf("done.final: %+v", state.LatestOutput("done"))
	}
}
