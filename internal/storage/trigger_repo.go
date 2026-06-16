// TriggerRepo persists the cron and webhook trigger configurations attached
// to a workflow. The scheduler reads enabled cron rows at boot, the webhook
// handler resolves single rows by id, and the API handler manages the CRUD
// lifecycle. config is opaque jsonb to keep new trigger kinds additive.
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

type Trigger struct {
	ID          string
	WorkflowID  string
	Kind        string // "cron" | "webhook"
	Config      json.RawMessage
	Enabled     bool
	LastFiredAt *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type TriggerRepo struct {
	pool *pgxpool.Pool
}

func NewTriggerRepo(pool *pgxpool.Pool) *TriggerRepo { return &TriggerRepo{pool: pool} }

func (r *TriggerRepo) Insert(ctx context.Context, t Trigger) error {
	if len(t.Config) == 0 {
		t.Config = json.RawMessage(`{}`)
	}
	const q = `
		INSERT INTO triggers (id, workflow_id, kind, config, enabled)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err := r.pool.Exec(ctx, q, t.ID, t.WorkflowID, t.Kind, []byte(t.Config), t.Enabled)
	if err != nil {
		return fmt.Errorf("storage: insert trigger: %w", err)
	}
	return nil
}

func (r *TriggerRepo) Get(ctx context.Context, id string) (Trigger, error) {
	const q = `
		SELECT id, workflow_id, kind, config, enabled, last_fired_at, created_at, updated_at
		FROM triggers WHERE id = $1
	`
	var t Trigger
	var cfg []byte
	err := r.pool.QueryRow(ctx, q, id).Scan(
		&t.ID, &t.WorkflowID, &t.Kind, &cfg, &t.Enabled, &t.LastFiredAt, &t.CreatedAt, &t.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Trigger{}, ErrNotFound
	}
	if err != nil {
		return Trigger{}, fmt.Errorf("storage: get trigger: %w", err)
	}
	t.Config = cfg
	return t, nil
}

func (r *TriggerRepo) ListByWorkflow(ctx context.Context, workflowID string) ([]Trigger, error) {
	const q = `
		SELECT id, workflow_id, kind, config, enabled, last_fired_at, created_at, updated_at
		FROM triggers WHERE workflow_id = $1 ORDER BY created_at
	`
	rows, err := r.pool.Query(ctx, q, workflowID)
	if err != nil {
		return nil, fmt.Errorf("storage: list triggers by workflow: %w", err)
	}
	defer rows.Close()
	return collectTriggers(rows)
}

func (r *TriggerRepo) ListEnabledByKind(ctx context.Context, kind string) ([]Trigger, error) {
	const q = `
		SELECT id, workflow_id, kind, config, enabled, last_fired_at, created_at, updated_at
		FROM triggers WHERE enabled = true AND kind = $1 ORDER BY created_at
	`
	rows, err := r.pool.Query(ctx, q, kind)
	if err != nil {
		return nil, fmt.Errorf("storage: list enabled triggers: %w", err)
	}
	defer rows.Close()
	return collectTriggers(rows)
}

func (r *TriggerRepo) UpdateConfig(ctx context.Context, id string, cfg json.RawMessage, enabled bool) error {
	if len(cfg) == 0 {
		cfg = json.RawMessage(`{}`)
	}
	const q = `UPDATE triggers SET config = $2, enabled = $3 WHERE id = $1`
	tag, err := r.pool.Exec(ctx, q, id, []byte(cfg), enabled)
	if err != nil {
		return fmt.Errorf("storage: update trigger: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *TriggerRepo) Delete(ctx context.Context, id string) error {
	const q = `DELETE FROM triggers WHERE id = $1`
	tag, err := r.pool.Exec(ctx, q, id)
	if err != nil {
		return fmt.Errorf("storage: delete trigger: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *TriggerRepo) TouchLastFired(ctx context.Context, id string) error {
	const q = `UPDATE triggers SET last_fired_at = now() WHERE id = $1`
	tag, err := r.pool.Exec(ctx, q, id)
	if err != nil {
		return fmt.Errorf("storage: touch trigger: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func collectTriggers(rows pgx.Rows) ([]Trigger, error) {
	var out []Trigger
	for rows.Next() {
		var t Trigger
		var cfg []byte
		if err := rows.Scan(&t.ID, &t.WorkflowID, &t.Kind, &cfg, &t.Enabled,
			&t.LastFiredAt, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, fmt.Errorf("storage: scan trigger: %w", err)
		}
		t.Config = cfg
		out = append(out, t)
	}
	return out, rows.Err()
}
