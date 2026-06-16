package storage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type WorkflowRun struct {
	ID              string
	WorkflowID      string
	WorkflowVersion int
	Status          string
	TriggerKind     string
	TriggerID       *string
	TriggerPayload  json.RawMessage
	ParentRunID     *string
	Error           string
	StartedAt       *time.Time
	FinishedAt      *time.Time
	CreatedAt       time.Time
}

// RunFilter constrains ListForWorkflow. All fields are optional; an empty
// filter returns the most recent page. Cursor is opaque to callers and is
// only meaningful when fed back into a subsequent ListForWorkflow call.
type RunFilter struct {
	Status string
	From   *time.Time
	To     *time.Time
	Cursor string
	Limit  int
}

type NodeRun struct {
	ID            string
	WorkflowRunID string
	NodeID        string
	Iteration     int
	Status        string
	Input         json.RawMessage
	Output        json.RawMessage
	Error         json.RawMessage
	Attempts      int
	StartedAt     *time.Time
	FinishedAt    *time.Time
	DurationMs    *int
}

type NodeRunUpdate struct {
	Status     string
	Input      json.RawMessage
	Output     json.RawMessage
	Error      json.RawMessage
	Attempts   int
	StartedAt  *time.Time
	FinishedAt *time.Time
	DurationMs *int
}

type WorkflowRunRepo struct {
	pool *pgxpool.Pool
}

func NewWorkflowRunRepo(pool *pgxpool.Pool) *WorkflowRunRepo { return &WorkflowRunRepo{pool: pool} }

func (r *WorkflowRunRepo) NewRun(ctx context.Context, run WorkflowRun) error {
	if len(run.TriggerPayload) == 0 {
		run.TriggerPayload = json.RawMessage(`{}`)
	}
	const q = `
		INSERT INTO workflow_runs
		  (id, workflow_id, workflow_version, status, trigger_kind,
		   trigger_id, trigger_payload, parent_run_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := r.pool.Exec(ctx, q,
		run.ID, run.WorkflowID, run.WorkflowVersion, run.Status, run.TriggerKind,
		run.TriggerID, run.TriggerPayload, run.ParentRunID,
	)
	if err != nil {
		return fmt.Errorf("storage: new run: %w", err)
	}
	return nil
}

func (r *WorkflowRunRepo) Get(ctx context.Context, id string) (WorkflowRun, error) {
	const q = `
		SELECT id, workflow_id, workflow_version, status, trigger_kind,
		       trigger_id, COALESCE(trigger_payload, '{}'::jsonb), parent_run_id,
		       COALESCE(error, ''),
		       started_at, finished_at, created_at
		FROM workflow_runs WHERE id = $1
	`
	var run WorkflowRun
	err := r.pool.QueryRow(ctx, q, id).Scan(
		&run.ID, &run.WorkflowID, &run.WorkflowVersion, &run.Status, &run.TriggerKind,
		&run.TriggerID, &run.TriggerPayload, &run.ParentRunID,
		&run.Error, &run.StartedAt, &run.FinishedAt, &run.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return WorkflowRun{}, ErrNotFound
	}
	if err != nil {
		return WorkflowRun{}, fmt.Errorf("storage: get run: %w", err)
	}
	return run, nil
}

func (r *WorkflowRunRepo) UpdateRunStatus(ctx context.Context, id, status, errMsg string, startedAt, finishedAt *time.Time) error {
	const q = `
		UPDATE workflow_runs
		SET status = $2,
		    error = NULLIF($3, ''),
		    started_at = COALESCE($4, started_at),
		    finished_at = COALESCE($5, finished_at)
		WHERE id = $1
	`
	_, err := r.pool.Exec(ctx, q, id, status, errMsg, startedAt, finishedAt)
	if err != nil {
		return fmt.Errorf("storage: update run: %w", err)
	}
	return nil
}

func (r *WorkflowRunRepo) InsertNodeRun(ctx context.Context, nr NodeRun) error {
	const q = `
		INSERT INTO node_runs
		  (id, workflow_run_id, node_id, iteration, status, input, output, error, attempts,
		   started_at, finished_at, duration_ms)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`
	_, err := r.pool.Exec(ctx, q,
		nr.ID, nr.WorkflowRunID, nr.NodeID, nr.Iteration, nr.Status,
		nr.Input, nr.Output, nr.Error, nr.Attempts,
		nr.StartedAt, nr.FinishedAt, nr.DurationMs,
	)
	if err != nil {
		return fmt.Errorf("storage: insert node run: %w", err)
	}
	return nil
}

func (r *WorkflowRunRepo) UpdateNodeRun(ctx context.Context, id string, upd NodeRunUpdate) error {
	const q = `
		UPDATE node_runs
		SET status = $2,
		    input  = COALESCE($3, input),
		    output = COALESCE($4, output),
		    error  = COALESCE($5, error),
		    attempts = $6,
		    started_at  = COALESCE($7, started_at),
		    finished_at = COALESCE($8, finished_at),
		    duration_ms = COALESCE($9, duration_ms)
		WHERE id = $1
	`
	_, err := r.pool.Exec(ctx, q,
		id, upd.Status, upd.Input, upd.Output, upd.Error, upd.Attempts,
		upd.StartedAt, upd.FinishedAt, upd.DurationMs,
	)
	if err != nil {
		return fmt.Errorf("storage: update node run: %w", err)
	}
	return nil
}

// ListForWorkflow returns runs for a workflow ordered newest first, optionally
// filtered by status and a created_at range. Pagination uses a (created_at,
// id) tuple cursor so ties on the timestamp do not cause rows to be skipped
// or duplicated when the page boundary lands on simultaneous inserts.
//
// The returned cursor is empty when the caller has reached the end. A
// non-empty cursor is only set when the page is full; partial pages signal
// exhaustion.
func (r *WorkflowRunRepo) ListForWorkflow(ctx context.Context, workflowID string, f RunFilter) ([]WorkflowRun, string, error) {
	limit := f.Limit
	if limit <= 0 {
		limit = 50
	}

	args := []any{workflowID}
	var where []string
	where = append(where, "workflow_id = $1")
	if f.Status != "" {
		args = append(args, f.Status)
		where = append(where, fmt.Sprintf("status = $%d", len(args)))
	}
	if f.From != nil {
		args = append(args, *f.From)
		where = append(where, fmt.Sprintf("created_at >= $%d", len(args)))
	}
	if f.To != nil {
		args = append(args, *f.To)
		where = append(where, fmt.Sprintf("created_at <= $%d", len(args)))
	}
	if f.Cursor != "" {
		cAt, cID, err := decodeRunCursor(f.Cursor)
		if err != nil {
			return nil, "", fmt.Errorf("storage: list runs: %w", err)
		}
		args = append(args, cAt)
		atIdx := len(args)
		args = append(args, cID)
		idIdx := len(args)
		where = append(where, fmt.Sprintf(
			"(created_at, id) < ($%d, $%d)", atIdx, idIdx,
		))
	}
	args = append(args, limit)
	limitIdx := len(args)

	q := fmt.Sprintf(`
		SELECT id, workflow_id, workflow_version, status, trigger_kind,
		       trigger_id, COALESCE(trigger_payload, '{}'::jsonb), parent_run_id,
		       COALESCE(error, ''),
		       started_at, finished_at, created_at
		FROM workflow_runs
		WHERE %s
		ORDER BY created_at DESC, id DESC
		LIMIT $%d
	`, strings.Join(where, " AND "), limitIdx)

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, "", fmt.Errorf("storage: list runs: %w", err)
	}
	defer rows.Close()

	out := make([]WorkflowRun, 0, limit)
	for rows.Next() {
		var run WorkflowRun
		if err := rows.Scan(
			&run.ID, &run.WorkflowID, &run.WorkflowVersion, &run.Status, &run.TriggerKind,
			&run.TriggerID, &run.TriggerPayload, &run.ParentRunID,
			&run.Error, &run.StartedAt, &run.FinishedAt, &run.CreatedAt,
		); err != nil {
			return nil, "", fmt.Errorf("storage: scan run: %w", err)
		}
		out = append(out, run)
	}
	if err := rows.Err(); err != nil {
		return nil, "", fmt.Errorf("storage: list runs: %w", err)
	}

	var nextCursor string
	if len(out) == limit {
		last := out[len(out)-1]
		nextCursor = encodeRunCursor(last.CreatedAt, last.ID)
	}
	return out, nextCursor, nil
}

// GetWithNodes returns a single run plus all of its node_runs in one call.
// The viewer and replay paths both need this exact shape.
func (r *WorkflowRunRepo) GetWithNodes(ctx context.Context, runID string) (WorkflowRun, []NodeRun, error) {
	run, err := r.Get(ctx, runID)
	if err != nil {
		return WorkflowRun{}, nil, err
	}
	nodes, err := r.ListNodeRuns(ctx, runID)
	if err != nil {
		return WorkflowRun{}, nil, err
	}
	return run, nodes, nil
}

// encodeRunCursor packs (created_at, id) into "<unix_nanos>:<id>". UnixNano
// is timezone-independent and preserves microsecond ordering coming back
// out of Postgres.
func encodeRunCursor(at time.Time, id string) string {
	return strconv.FormatInt(at.UTC().UnixNano(), 10) + ":" + id
}

func decodeRunCursor(cur string) (time.Time, string, error) {
	i := strings.IndexByte(cur, ':')
	if i < 1 || i == len(cur)-1 {
		return time.Time{}, "", fmt.Errorf("invalid cursor")
	}
	ns, err := strconv.ParseInt(cur[:i], 10, 64)
	if err != nil {
		return time.Time{}, "", fmt.Errorf("invalid cursor timestamp")
	}
	return time.Unix(0, ns).UTC(), cur[i+1:], nil
}

func (r *WorkflowRunRepo) ListNodeRuns(ctx context.Context, runID string) ([]NodeRun, error) {
	const q = `
		SELECT id, workflow_run_id, node_id, iteration, status,
		       COALESCE(input,'null'::jsonb), COALESCE(output,'null'::jsonb),
		       COALESCE(error,'null'::jsonb), attempts, started_at, finished_at, duration_ms
		FROM node_runs WHERE workflow_run_id = $1
		ORDER BY started_at NULLS LAST, id
	`
	rows, err := r.pool.Query(ctx, q, runID)
	if err != nil {
		return nil, fmt.Errorf("storage: list node runs: %w", err)
	}
	defer rows.Close()
	var out []NodeRun
	for rows.Next() {
		var nr NodeRun
		if err := rows.Scan(&nr.ID, &nr.WorkflowRunID, &nr.NodeID, &nr.Iteration, &nr.Status,
			&nr.Input, &nr.Output, &nr.Error, &nr.Attempts,
			&nr.StartedAt, &nr.FinishedAt, &nr.DurationMs); err != nil {
			return nil, fmt.Errorf("storage: scan node run: %w", err)
		}
		out = append(out, nr)
	}
	return out, rows.Err()
}
