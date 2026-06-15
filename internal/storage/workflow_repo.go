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

type Workflow struct {
	ID                     string
	WorkspaceID            string
	Name                   string
	Status                 string
	CurrentVersion         int
	DefaultLLMCredentialID *string
	CreatedAt              time.Time
	UpdatedAt              time.Time
}

type WorkflowVersion struct {
	ID              string
	WorkflowID      string
	Version         int
	Definition      json.RawMessage
	Message         string
	CreatedByUserID *string
	CreatedAt       time.Time
}

type WorkflowRepo struct {
	pool *pgxpool.Pool
}

func NewWorkflowRepo(pool *pgxpool.Pool) *WorkflowRepo { return &WorkflowRepo{pool: pool} }

func (r *WorkflowRepo) Insert(ctx context.Context, w Workflow) error {
	if w.CurrentVersion == 0 {
		w.CurrentVersion = 1
	}
	if w.Status == "" {
		w.Status = "draft"
	}
	const q = `
		INSERT INTO workflows (id, workspace_id, name, status, current_version, default_llm_credential_id)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := r.pool.Exec(ctx, q, w.ID, w.WorkspaceID, w.Name, w.Status, w.CurrentVersion, w.DefaultLLMCredentialID)
	if err != nil {
		return fmt.Errorf("storage: insert workflow: %w", err)
	}
	return nil
}

func (r *WorkflowRepo) Get(ctx context.Context, id string) (Workflow, error) {
	const q = `
		SELECT id, workspace_id, name, status, current_version, default_llm_credential_id,
		       created_at, updated_at
		FROM workflows WHERE id = $1
	`
	var w Workflow
	err := r.pool.QueryRow(ctx, q, id).Scan(
		&w.ID, &w.WorkspaceID, &w.Name, &w.Status, &w.CurrentVersion, &w.DefaultLLMCredentialID,
		&w.CreatedAt, &w.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Workflow{}, ErrNotFound
	}
	if err != nil {
		return Workflow{}, fmt.Errorf("storage: get workflow: %w", err)
	}
	return w, nil
}

func (r *WorkflowRepo) ListByWorkspace(ctx context.Context, workspaceID string) ([]Workflow, error) {
	const q = `
		SELECT id, workspace_id, name, status, current_version, default_llm_credential_id,
		       created_at, updated_at
		FROM workflows WHERE workspace_id = $1 ORDER BY created_at DESC
	`
	rows, err := r.pool.Query(ctx, q, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("storage: list workflows: %w", err)
	}
	defer rows.Close()
	var out []Workflow
	for rows.Next() {
		var w Workflow
		if err := rows.Scan(&w.ID, &w.WorkspaceID, &w.Name, &w.Status, &w.CurrentVersion,
			&w.DefaultLLMCredentialID, &w.CreatedAt, &w.UpdatedAt); err != nil {
			return nil, fmt.Errorf("storage: scan workflow: %w", err)
		}
		out = append(out, w)
	}
	return out, rows.Err()
}

func (r *WorkflowRepo) SaveVersion(ctx context.Context, v WorkflowVersion) error {
	const q = `
		INSERT INTO workflow_versions (id, workflow_id, version, definition, message, created_by_user_id)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := r.pool.Exec(ctx, q, v.ID, v.WorkflowID, v.Version, v.Definition, v.Message, v.CreatedByUserID)
	if err != nil {
		return fmt.Errorf("storage: save version: %w", err)
	}
	return nil
}

func (r *WorkflowRepo) GetVersion(ctx context.Context, workflowID string, version int) (WorkflowVersion, error) {
	const q = `
		SELECT id, workflow_id, version, definition, message, created_by_user_id, created_at
		FROM workflow_versions WHERE workflow_id = $1 AND version = $2
	`
	var v WorkflowVersion
	err := r.pool.QueryRow(ctx, q, workflowID, version).Scan(
		&v.ID, &v.WorkflowID, &v.Version, &v.Definition, &v.Message, &v.CreatedByUserID, &v.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return WorkflowVersion{}, ErrNotFound
	}
	if err != nil {
		return WorkflowVersion{}, fmt.Errorf("storage: get version: %w", err)
	}
	return v, nil
}
