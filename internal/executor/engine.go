package executor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/efekckk/flowgent/internal/expression"
	"github.com/efekckk/flowgent/internal/idgen"
	"github.com/efekckk/flowgent/internal/registry"
)

type RunOptions struct {
	TriggerKind    string
	TriggerPayload map[string]any
	// RunID is forwarded to the configured LogSink so log rows can be tied
	// back to a workflow_runs row. Optional: when empty the engine still
	// emits events, the sink just sees an empty run id.
	RunID string
}

type Engine struct {
	registry     *registry.Registry
	maxAttempts  int
	backoff      func(attempt int) time.Duration
	maxParallel  int
	credResolver CredentialResolver
	runStore     RunStore
	idgen        func() string
	logSink      LogSink
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
		idgen:       idgen.NewRun,
		logSink:     noopSink{},
	}
	for _, opt := range opts {
		opt(e)
	}
	return e
}

// WithRunIDGenerator overrides the generator used by RunFromReplay so tests
// can produce deterministic ids. Defaults to idgen.NewRun.
func WithRunIDGenerator(fn func() string) Option {
	return func(e *Engine) {
		if fn != nil {
			e.idgen = fn
		}
	}
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
				// Surface the names of upstreams we are still waiting on so a
				// run viewer can explain why a merge node hasn't fired yet.
				missing := pendingUpstreams(wf, n.ID, state)
				e.logSink.Append(ctx, opts.RunID, n.ID, "info",
					fmt.Sprintf("%s: waiting for %v", n.ID, missing))
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

// ErrNoRunStore is returned by RunFromReplay when the engine was created
// without a RunStore. Replays require persistence; production wiring must
// pass WithRunStore.
var ErrNoRunStore = errors.New("executor: replay requires a run store")

// RunFromReplay clones the given parent run as a fresh execution. The
// caller supplies the trigger payload (already extracted from the parent
// or overridden by the user); the engine then:
//
//  1. resolves the workflow to its current definition,
//  2. inserts a new workflow_runs row marked 'replay' with parent_run_id,
//  3. dispatches the workflow exactly the way Run does,
//  4. persists node_runs + the final status,
//  5. returns the new run id.
//
// The new run's trigger_kind is "replay" so the run history view can
// distinguish replays at a glance without losing the original trigger
// metadata, which is preserved on the parent row.
func (e *Engine) RunFromReplay(ctx context.Context, workflowID, parentRunID string, payload map[string]any) (string, error) {
	if e.runStore == nil {
		return "", ErrNoRunStore
	}

	version, wf, err := e.runStore.LoadWorkflowForRun(ctx, workflowID)
	if err != nil {
		return "", fmt.Errorf("executor: replay load workflow: %w", err)
	}
	if wf == nil {
		return "", fmt.Errorf("executor: replay: workflow %q has no definition", workflowID)
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("executor: replay marshal payload: %w", err)
	}
	if len(payloadBytes) == 0 || string(payloadBytes) == "null" {
		payloadBytes = json.RawMessage(`{}`)
	}

	runID := e.idgen()
	startedAt := time.Now().UTC()
	parent := parentRunID
	if err := e.runStore.InsertRun(ctx, InsertRunParams{
		ID:              runID,
		WorkflowID:      workflowID,
		WorkflowVersion: version,
		TriggerKind:     "replay",
		TriggerPayload:  payloadBytes,
		ParentRunID:     &parent,
		StartedAt:       startedAt,
	}); err != nil {
		return "", fmt.Errorf("executor: replay insert run: %w", err)
	}

	// Once the row exists, every exit path from here must terminate it. If
	// PersistRun never runs (panic, parent ctx already cancelled, etc.)
	// the defer below stamps the row as failed via a fresh background
	// context so the run viewer never shows a permanently-running ghost.
	persisted := false
	defer func() {
		if persisted {
			return
		}
		bgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = e.runStore.FailRun(bgCtx, runID, "engine exited before persisting run", time.Now().UTC())
	}()

	state, runErr := e.Run(ctx, wf, RunOptions{
		TriggerKind:    "replay",
		TriggerPayload: payload,
		RunID:          runID,
	})
	finishedAt := time.Now().UTC()
	status, statusErr := state.RunStatus()
	errMsg := statusErr
	if runErr != nil && errMsg == "" {
		errMsg = runErr.Error()
	}
	if err := e.runStore.PersistRun(ctx, runID, wf, state, status, errMsg, startedAt, finishedAt); err != nil {
		return runID, fmt.Errorf("executor: replay persist: %w", err)
	}
	persisted = true
	return runID, nil
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

	// Emit "started" before resolving inputs so a viewer sees the node light
	// up immediately. Tool slug + node id only — never the resolved input map
	// (which can contain a resolved "__credential").
	e.logSink.Append(ctx, opts.RunID, node.ID, "info",
		fmt.Sprintf("%s: started", node.Tool))
	state.SetStatus(node.ID, "running")
	startedAt := time.Now()
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
		// Retry-with-backoff: report the classified error verbatim; tools are
		// responsible (per M7) for redacting secrets in their error messages.
		e.logSink.Append(ctx, opts.RunID, node.ID, "warn",
			fmt.Sprintf("%s: retry %d/%d: %v", node.Tool, attempts, e.maxAttempts, execErr))
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
			// Error-port routed: not a run-level failure, just a branch swap.
			e.logSink.Append(ctx, opts.RunID, node.ID, "info",
				fmt.Sprintf("%s: error routed via error port: %v", node.Tool, execErr))
			state.RecordFailure(node.ID, 0, attempts, execErr.Error())
			state.RecordOutput(node.ID, 0, map[string]any{"error": execErr.Error()}, "error")
			return "error", nil
		}
		// Final failure — error message only, never the input map.
		e.logSink.Append(ctx, opts.RunID, node.ID, "error",
			fmt.Sprintf("%s: failed: %v", node.Tool, execErr))
		state.RecordFailure(node.ID, 0, attempts, execErr.Error())
		state.SetRunStatus("failed", execErr.Error())
		return "", execErr
	}
	// Success — log elapsed time and the number of top-level output keys; the
	// output content itself (which may include user-fetched bodies) is kept
	// out of the log message.
	elapsed := time.Since(startedAt).Milliseconds()
	e.logSink.Append(ctx, opts.RunID, node.ID, "info",
		fmt.Sprintf("%s: succeeded in %dms (%d output keys)", node.Tool, elapsed, len(res.Output)))
	state.RecordOutput(node.ID, 0, res.Output, res.Port)
	return res.Port, nil
}
