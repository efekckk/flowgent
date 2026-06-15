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

func (e *Engine) Run(ctx context.Context, wf *Workflow, opts RunOptions) (*RunState, error) {
	state := NewRunState(len(wf.Nodes))
	for _, n := range wf.Nodes {
		state.SetStatus(n.ID, "pending")
	}

	pending := wf.EntryNodes()
	if len(pending) == 0 && len(wf.Nodes) > 0 {
		state.SetRunStatus("failed", "no entry nodes (cycle?)")
		return state, fmt.Errorf("executor: no entry nodes")
	}

	visited := make(map[string]bool)
	for len(pending) > 0 {
		node := pending[0]
		pending = pending[1:]
		if visited[node.ID] || state.Status(node.ID) != "pending" {
			continue
		}
		visited[node.ID] = true

		port, err := e.executeNode(ctx, &node, state, opts)
		if err != nil {
			return state, err
		}

		for _, edge := range wf.Edges {
			if edge.From != node.ID {
				continue
			}
			next, ok := wf.NodeByID(edge.To)
			if !ok {
				continue
			}
			if edge.FromPort == port {
				pending = append(pending, next)
			} else {
				state.SetStatus(next.ID, "skipped")
			}
		}
	}

	state.SetRunStatus("succeeded", "")
	return state, nil
}

func (e *Engine) executeNode(ctx context.Context, node *Node, state *RunState, opts RunOptions) (string, error) {
	exec, ok := e.registry.Get(node.Tool)
	if !ok {
		state.SetStatus(node.ID, "failed")
		state.SetRunStatus("failed", fmt.Sprintf("unknown tool %q", node.Tool))
		return "", fmt.Errorf("executor: unknown tool %q", node.Tool)
	}

	state.SetStatus(node.ID, "running")
	evalCtx := expression.EvalContext{
		Trigger: opts.TriggerPayload,
		Nodes:   state.LatestOutputsMap(),
	}
	resolved, err := ResolveInputs(node.Params, node.ID, evalCtx, state)
	if err != nil {
		state.SetStatus(node.ID, "failed")
		state.SetRunStatus("failed", err.Error())
		return "", err
	}
	state.SetInput(node.ID, resolved)

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
		state.RecordFailure(node.ID, 0, attempts, execErr.Error())
		state.SetRunStatus("failed", execErr.Error())
		return "", execErr
	}
	state.RecordOutput(node.ID, 0, res.Output, res.Port)
	return res.Port, nil
}
