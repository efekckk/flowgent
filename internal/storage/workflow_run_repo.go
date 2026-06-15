package storage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
	TriggerPayload  json.RawMessage
	Error           string
	StartedAt       *time.Time
	FinishedAt      *time.Time
	CreatedAt       time.Time
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
	const q = `
		INSERT INTO workflow_runs (id, workflow_id, workflow_version, status, trigger_kind, trigger_payload)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := r.pool.Exec(ctx, q,
		run.ID, run.WorkflowID, run.WorkflowVersion, run.Status, run.TriggerKind, run.TriggerPayload,
	)
	if err != nil {
		return fmt.Errorf("storage: new run: %w", err)
	}
	return nil
}

func (r *WorkflowRunRepo) Get(ctx context.Context, id string) (WorkflowRun, error) {
	const q = `
		SELECT id, workflow_id, workflow_version, status, trigger_kind,
		       COALESCE(trigger_payload, 'null'::jsonb), COALESCE(error, ''),
		       started_at, finished_at, created_at
		FROM workflow_runs WHERE id = $1
	`
	var run WorkflowRun
	err := r.pool.QueryRow(ctx, q, id).Scan(
		&run.ID, &run.WorkflowID, &run.WorkflowVersion, &run.Status, &run.TriggerKind,
		&run.TriggerPayload, &run.Error, &run.StartedAt, &run.FinishedAt, &run.CreatedAt,
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
