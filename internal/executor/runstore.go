package executor

import (
	"context"
	"encoding/json"
	"time"
)

// RunStore is the minimal storage contract the engine needs to fan out a
// replay run on its own. It is intentionally narrow so the real
// implementation in package storage can satisfy it via a thin adapter in
// cmd/flowgent. Keeping it inside executor keeps the import graph one-way
// (api/cmd -> executor -> registry) and avoids pulling pgx into the engine
// build.
type RunStore interface {
	// LoadWorkflowForRun resolves a workflow id to (workflow_version,
	// parsed Workflow definition). The engine treats the version field
	// as opaque metadata to persist on the run row.
	LoadWorkflowForRun(ctx context.Context, workflowID string) (version int, def *Workflow, err error)

	// InsertRun materialises a workflow_runs row in 'running' state. The
	// engine calls this exactly once at the start of RunFromReplay.
	InsertRun(ctx context.Context, params InsertRunParams) error

	// PersistRun is called after the workflow has finished. The engine
	// hands back the final RunState plus the resolved final status, and
	// the implementation walks node_runs + updates the workflow_runs row.
	PersistRun(ctx context.Context, runID string, wf *Workflow, state *RunState,
		status, errMsg string, startedAt, finishedAt time.Time) error

	// GetTriggerPayload returns the parent run's stored trigger_payload
	// for cloning during a replay. Returned bytes are jsonb-shaped (may
	// be 'null' or '{}').
	GetTriggerPayload(ctx context.Context, runID string) (json.RawMessage, error)
}

// InsertRunParams is the run-row contract for RunStore.InsertRun. ParentRunID
// is nil for first-class runs and set for replays; TriggerPayload is the
// jsonb-encoded payload to persist verbatim.
type InsertRunParams struct {
	ID              string
	WorkflowID      string
	WorkflowVersion int
	TriggerKind     string
	TriggerPayload  json.RawMessage
	ParentRunID     *string
	StartedAt       time.Time
}

// WithRunStore attaches a RunStore so the engine can drive replay
// end-to-end without the caller threading repos through.
func WithRunStore(s RunStore) Option {
	return func(e *Engine) { e.runStore = s }
}
