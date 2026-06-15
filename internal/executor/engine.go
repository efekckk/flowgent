package executor

import (
	"context"
	"fmt"
	"time"

	"github.com/efekckk/flowgent/internal/expression"
	"github.com/efekckk/flowgent/internal/registry"
)

type RunOptions struct {
	TriggerKind    string
	TriggerPayload map[string]any
}

type RunState struct {
	Status      string
	Error       string
	NodeStatus  map[string]string
	NodeOutputs map[string]map[string]any
	NodeInputs  map[string]map[string]any
	NodePort    map[string]string
}

type Engine struct {
	registry    *registry.Registry
	maxAttempts int
	backoff     func(attempt int) time.Duration
}

type Option func(*Engine)

func WithMaxAttempts(n int) Option {
	return func(e *Engine) { e.maxAttempts = n }
}

func WithBackoff(fn func(attempt int) time.Duration) Option {
	return func(e *Engine) { e.backoff = fn }
}

func NewEngine(reg *registry.Registry, opts ...Option) *Engine {
	e := &Engine{
		registry:    reg,
		maxAttempts: 3,
		backoff:     defaultBackoff,
	}
	for _, opt := range opts {
		opt(e)
	}
	return e
}

func defaultBackoff(attempt int) time.Duration {
	base := 500 * time.Millisecond
	for i := 1; i < attempt; i++ {
		base *= 2
	}
	return base
}

func (e *Engine) Run(ctx context.Context, wf *Workflow, opts RunOptions) (RunState, error) {
	state := RunState{
		Status:      "running",
		NodeStatus:  make(map[string]string, len(wf.Nodes)),
		NodeOutputs: make(map[string]map[string]any, len(wf.Nodes)),
		NodeInputs:  make(map[string]map[string]any, len(wf.Nodes)),
		NodePort:    make(map[string]string, len(wf.Nodes)),
	}
	for _, n := range wf.Nodes {
		state.NodeStatus[n.ID] = "pending"
	}

	pending := wf.EntryNodes()
	if len(pending) == 0 && len(wf.Nodes) > 0 {
		state.Status = "failed"
		state.Error = "no entry nodes (cycle?)"
		return state, fmt.Errorf("executor: no entry nodes")
	}

	visited := make(map[string]bool)
	for len(pending) > 0 {
		node := pending[0]
		pending = pending[1:]
		if visited[node.ID] || state.NodeStatus[node.ID] != "pending" {
			continue
		}
		visited[node.ID] = true

		exec, ok := e.registry.Get(node.Tool)
		if !ok {
			state.NodeStatus[node.ID] = "failed"
			state.Status = "failed"
			state.Error = fmt.Sprintf("unknown tool %q", node.Tool)
			return state, fmt.Errorf("executor: unknown tool %q", node.Tool)
		}

		state.NodeStatus[node.ID] = "running"

		evalCtx := expression.EvalContext{
			Trigger: opts.TriggerPayload,
			Nodes:   state.NodeOutputs,
		}
		resolved, err := ResolveInputs(node.Params, node.ID, evalCtx, &state)
		if err != nil {
			state.NodeStatus[node.ID] = "failed"
			state.Status = "failed"
			state.Error = err.Error()
			return state, err
		}
		state.NodeInputs[node.ID] = resolved

		var res registry.ExecuteResult
		var execErr error
		attempts := 0
		for attempts < e.maxAttempts {
			attempts++
			res, execErr = exec.Execute(ctx, resolved)
			if execErr == nil {
				break
			}
			if !IsRetryable(execErr) || attempts >= e.maxAttempts {
				break
			}
			d := e.backoff(attempts)
			if d > 0 {
				select {
				case <-time.After(d):
				case <-ctx.Done():
					execErr = ctx.Err()
				}
				if ctx.Err() != nil {
					break
				}
			}
		}
		if execErr != nil {
			state.NodeStatus[node.ID] = "failed"
			state.Status = "failed"
			state.Error = execErr.Error()
			return state, execErr
		}
		state.NodeStatus[node.ID] = "succeeded"
		state.NodeOutputs[node.ID] = res.Output
		state.NodePort[node.ID] = res.Port

		for _, edge := range wf.Edges {
			if edge.From != node.ID {
				continue
			}
			next, ok := wf.NodeByID(edge.To)
			if !ok {
				continue
			}
			if edge.FromPort == res.Port {
				pending = append(pending, next)
			} else {
				state.NodeStatus[next.ID] = "skipped"
			}
		}
	}

	state.Status = "succeeded"
	return state, nil
}
