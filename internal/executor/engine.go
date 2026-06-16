package executor

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/efekckk/flowgent/internal/expression"
	"github.com/efekckk/flowgent/internal/registry"
)

type RunOptions struct {
	TriggerKind    string
	TriggerPayload map[string]any
}

type Engine struct {
	registry     *registry.Registry
	maxAttempts  int
	backoff      func(attempt int) time.Duration
	maxParallel  int
	credResolver CredentialResolver
}

type Option func(*Engine)

func WithMaxAttempts(n int) Option {
	return func(e *Engine) { e.maxAttempts = n }
}

func WithBackoff(fn func(attempt int) time.Duration) Option {
	return func(e *Engine) { e.backoff = fn }
}

func WithMaxParallel(n int) Option {
	return func(e *Engine) {
		if n > 0 {
			e.maxParallel = n
		}
	}
}

func NewEngine(reg *registry.Registry, opts ...Option) *Engine {
	e := &Engine{
		registry:    reg,
		maxAttempts: 3,
		backoff:     defaultBackoff,
		maxParallel: 8,
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
		batch := make([]*Node, 0, len(pending))
		seen := make(map[string]bool, len(pending))
		preserved := make([]Node, 0)
		for _, n := range pending {
			if visited[n.ID] || seen[n.ID] || state.Status(n.ID) != "pending" {
				continue
			}
			if IsMergeNode(&n) && !AllUpstreamSucceeded(wf, n.ID, state) {
				preserved = append(preserved, n)
				continue
			}
			seen[n.ID] = true
			nn := n
			batch = append(batch, &nn)
		}
		pending = pending[:0]
		pending = append(pending, preserved...)
		if len(batch) == 0 {
			break
		}

		sem := make(chan struct{}, e.maxParallel)
		type result struct {
			node *Node
			port string
			err  error
		}
		results := make([]result, len(batch))
		var wg sync.WaitGroup
		for i, node := range batch {
			i, node := i, node
			visited[node.ID] = true
			wg.Add(1)
			sem <- struct{}{}
			go func() {
				defer wg.Done()
				defer func() { <-sem }()
				var port string
				var err error
				if IsLoopNode(node) {
					port, err = e.runLoop(ctx, wf, node, state, opts)
				} else {
					port, err = e.executeNode(ctx, wf, node, state, opts)
				}
				results[i] = result{node: node, port: port, err: err}
			}()
		}
		wg.Wait()

		for _, r := range results {
			if r.err != nil {
				return state, r.err
			}
		}

		for _, r := range results {
			for _, edge := range wf.Edges {
				if edge.From != r.node.ID {
					continue
				}
				next, ok := wf.NodeByID(edge.To)
				if !ok {
					continue
				}
				if edge.FromPort == r.port {
					pending = append(pending, next)
				} else {
					if state.Status(next.ID) == "pending" {
						state.SetStatus(next.ID, "skipped")
					}
				}
			}
		}
	}

	state.SetRunStatus("succeeded", "")
	return state, nil
}

func (e *Engine) executeNode(ctx context.Context, wf *Workflow, node *Node, state *RunState, opts RunOptions) (string, error) {
	exec, ok := e.registry.Get(node.Tool)
	if !ok {
		state.SetStatus(node.ID, "failed")
		state.SetRunStatus("failed", fmt.Sprintf("unknown tool %q", node.Tool))
		return "", fmt.Errorf("executor: unknown tool %q", node.Tool)
	}

	// Merge nodes get their `inputs` materialised by the engine right before
	// resolution, replacing the magic placeholder.
	if IsMergeNode(node) {
		upstreams := CollectUpstreamOutputs(wf, node.ID, state)
		if node.Params == nil {
			node.Params = map[string]any{}
		}
		node.Params["inputs"] = upstreams
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

	if node.Credential != "" {
		if e.credResolver == nil {
			state.SetStatus(node.ID, "failed")
			err := fmt.Errorf("executor: node %q references credential %q but no resolver configured", node.ID, node.Credential)
			state.SetRunStatus("failed", err.Error())
			return "", err
		}
		secret, err := e.credResolver.Resolve(ctx, node.Credential)
		if err != nil {
			state.SetStatus(node.ID, "failed")
			state.SetRunStatus("failed", err.Error())
			return "", fmt.Errorf("executor: resolve credential for node %q: %w", node.ID, err)
		}
		resolved["__credential"] = secret
	}

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
		hasErrorEdge := false
		for _, edge := range wf.Edges {
			if edge.From == node.ID && edge.FromPort == "error" {
				hasErrorEdge = true
				break
			}
		}
		if hasErrorEdge {
			state.RecordFailure(node.ID, 0, attempts, execErr.Error())
			state.RecordOutput(node.ID, 0, map[string]any{"error": execErr.Error()}, "error")
			return "error", nil
		}
		state.RecordFailure(node.ID, 0, attempts, execErr.Error())
		state.SetRunStatus("failed", execErr.Error())
		return "", execErr
	}
	state.RecordOutput(node.ID, 0, res.Output, res.Port)
	return res.Port, nil
}
