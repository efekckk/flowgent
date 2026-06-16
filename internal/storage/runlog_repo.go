package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// RunLog is a single log line emitted while a workflow_run was executing.
// NodeID is empty for run-level events (start, finish, replay link). All
// timestamps are server-assigned via DEFAULT now() to keep monotonic order
// stable across multiple writers.
type RunLog struct {
	ID      int64
	RunID   string
	NodeID  string
	Level   string
	Message string
	At      time.Time
}

// RunLogRepo keeps append-only per-run log entries with a generated tsvector
// column for full-text search. There is no edit/delete API at the repo
// layer; rows go away only when the parent workflow_run is deleted
// (CASCADE).
type RunLogRepo struct {
	pool *pgxpool.Pool
}

func NewRunLogRepo(pool *pgxpool.Pool) *RunLogRepo { return &RunLogRepo{pool: pool} }

// Append writes a single log line. node_id is normalised to NULL when the
// caller passes an empty string so the GROUP/JOIN paths stay simple.
func (r *RunLogRepo) Append(ctx context.Context, e RunLog) error {
	const q = `
		INSERT INTO run_logs (run_id, node_id, level, message)
		VALUES ($1, NULLIF($2, ''), $3, $4)
	`
	if _, err := r.pool.Exec(ctx, q, e.RunID, e.NodeID, e.Level, e.Message); err != nil {
		return fmt.Errorf("storage: append run log: %w", err)
	}
	return nil
}

// ListByRun returns log rows for one run, in insertion order. sinceID lets a
// polling consumer (the run viewer in the SPA) ask only for rows it hasn't
// seen yet without paying for an offset scan.
func (r *RunLogRepo) ListByRun(ctx context.Context, runID string, sinceID int64, limit int) ([]RunLog, error) {
	if limit <= 0 {
		limit = 200
	}
	const q = `
		SELECT id, run_id, COALESCE(node_id, ''), level, message, at
		FROM run_logs
		WHERE run_id = $1 AND id > $2
		ORDER BY id
		LIMIT $3
	`
	rows, err := r.pool.Query(ctx, q, runID, sinceID, limit)
	if err != nil {
		return nil, fmt.Errorf("storage: list run logs: %w", err)
	}
	defer rows.Close()
	out := make([]RunLog, 0, limit)
	for rows.Next() {
		var e RunLog
		if err := rows.Scan(&e.ID, &e.RunID, &e.NodeID, &e.Level, &e.Message, &e.At); err != nil {
			return nil, fmt.Errorf("storage: scan run log: %w", err)
		}
		out = append(out, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("storage: list run logs: %w", err)
	}
	return out, nil
}

// SearchHit is one full-text match returned from Search; the snippet is the
// ts_headline-decorated message excerpt suitable for direct rendering with
// «…» fragment markers.
type SearchHit struct {
	RunID      string    `json:"run_id"`
	WorkflowID string    `json:"workflow_id"`
	NodeID     string    `json:"node_id"`
	Message    string    `json:"message"`
	Snippet    string    `json:"snippet"`
	At         time.Time `json:"at"`
}

// Search runs a workspace-scoped full-text query over run_logs.message using
// the generated search_doc tsvector. Scoping is enforced by joining through
// workflow_runs -> workflows.workspace_id so a caller can only ever see
// logs they own.
func (r *RunLogRepo) Search(ctx context.Context, workspaceID, q string, limit int) ([]SearchHit, error) {
	if limit <= 0 {
		limit = 50
	}
	const sql = `
		SELECT l.run_id,
		       wr.workflow_id,
		       COALESCE(l.node_id, ''),
		       l.message,
		       ts_headline('english', l.message, plainto_tsquery('english', $2),
		                   'StartSel=«,StopSel=»,MaxFragments=2'),
		       l.at
		FROM run_logs l
		JOIN workflow_runs wr ON wr.id = l.run_id
		JOIN workflows w      ON w.id = wr.workflow_id
		WHERE w.workspace_id = $1
		  AND l.search_doc @@ plainto_tsquery('english', $2)
		ORDER BY l.at DESC
		LIMIT $3
	`
	rows, err := r.pool.Query(ctx, sql, workspaceID, q, limit)
	if err != nil {
		return nil, fmt.Errorf("storage: search run logs: %w", err)
	}
	defer rows.Close()
	out := make([]SearchHit, 0, limit)
	for rows.Next() {
		var h SearchHit
		if err := rows.Scan(&h.RunID, &h.WorkflowID, &h.NodeID, &h.Message, &h.Snippet, &h.At); err != nil {
			return nil, fmt.Errorf("storage: scan run log hit: %w", err)
		}
		out = append(out, h)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("storage: search run logs: %w", err)
	}
	return out, nil
}
