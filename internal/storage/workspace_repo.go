package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Workspace struct {
	ID          string
	OwnerUserID string
	Name        string
	CreatedAt   time.Time
}

type WorkspaceRepo struct {
	pool *pgxpool.Pool
}

func NewWorkspaceRepo(pool *pgxpool.Pool) *WorkspaceRepo { return &WorkspaceRepo{pool: pool} }

func (r *WorkspaceRepo) Insert(ctx context.Context, w Workspace) error {
	const q = `INSERT INTO workspaces (id, owner_user_id, name) VALUES ($1, $2, $3)`
	if _, err := r.pool.Exec(ctx, q, w.ID, w.OwnerUserID, w.Name); err != nil {
		return fmt.Errorf("storage: insert workspace: %w", err)
	}
	return nil
}

func (r *WorkspaceRepo) FindByOwner(ctx context.Context, userID string) ([]Workspace, error) {
	const q = `
		SELECT id, owner_user_id, name, created_at
		FROM workspaces WHERE owner_user_id = $1 ORDER BY created_at ASC
	`
	rows, err := r.pool.Query(ctx, q, userID)
	if err != nil {
		return nil, fmt.Errorf("storage: query workspaces: %w", err)
	}
	defer rows.Close()
	var out []Workspace
	for rows.Next() {
		var w Workspace
		if err := rows.Scan(&w.ID, &w.OwnerUserID, &w.Name, &w.CreatedAt); err != nil {
			return nil, fmt.Errorf("storage: scan workspace: %w", err)
		}
		out = append(out, w)
	}
	return out, rows.Err()
}
