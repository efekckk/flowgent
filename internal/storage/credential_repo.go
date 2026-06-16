package storage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Credential struct {
	ID          string
	WorkspaceID string
	Name        string
	Type        string
	Encrypted   []byte
	Meta        json.RawMessage
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type CredentialRepo struct {
	pool *pgxpool.Pool
}

func NewCredentialRepo(pool *pgxpool.Pool) *CredentialRepo { return &CredentialRepo{pool: pool} }

func (r *CredentialRepo) Insert(ctx context.Context, c Credential) error {
	if len(c.Meta) == 0 {
		c.Meta = json.RawMessage(`{}`)
	}
	const q = `
		INSERT INTO credentials (id, workspace_id, name, type, encrypted, meta)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := r.pool.Exec(ctx, q, c.ID, c.WorkspaceID, c.Name, c.Type, c.Encrypted, c.Meta)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return fmt.Errorf("%w: %s", ErrConflict, pgErr.ConstraintName)
		}
		return fmt.Errorf("storage: insert credential: %w", err)
	}
	return nil
}

func (r *CredentialRepo) Get(ctx context.Context, id string) (Credential, error) {
	const q = `
		SELECT id, workspace_id, name, type, encrypted, meta, created_at, updated_at
		FROM credentials WHERE id = $1
	`
	var c Credential
	err := r.pool.QueryRow(ctx, q, id).Scan(
		&c.ID, &c.WorkspaceID, &c.Name, &c.Type, &c.Encrypted, &c.Meta, &c.CreatedAt, &c.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Credential{}, ErrNotFound
	}
	if err != nil {
		return Credential{}, fmt.Errorf("storage: get credential: %w", err)
	}
	return c, nil
}

func (r *CredentialRepo) FindByName(ctx context.Context, workspaceID, name string) (Credential, error) {
	const q = `
		SELECT id, workspace_id, name, type, encrypted, meta, created_at, updated_at
		FROM credentials WHERE workspace_id = $1 AND name = $2
	`
	var c Credential
	err := r.pool.QueryRow(ctx, q, workspaceID, name).Scan(
		&c.ID, &c.WorkspaceID, &c.Name, &c.Type, &c.Encrypted, &c.Meta, &c.CreatedAt, &c.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Credential{}, ErrNotFound
	}
	if err != nil {
		return Credential{}, fmt.Errorf("storage: find credential: %w", err)
	}
	return c, nil
}

func (r *CredentialRepo) ListByWorkspace(ctx context.Context, workspaceID string) ([]Credential, error) {
	const q = `
		SELECT id, workspace_id, name, type, encrypted, meta, created_at, updated_at
		FROM credentials WHERE workspace_id = $1 ORDER BY created_at DESC
	`
	rows, err := r.pool.Query(ctx, q, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("storage: list credentials: %w", err)
	}
	defer rows.Close()
	var out []Credential
	for rows.Next() {
		var c Credential
		if err := rows.Scan(&c.ID, &c.WorkspaceID, &c.Name, &c.Type, &c.Encrypted, &c.Meta,
			&c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, fmt.Errorf("storage: scan credential: %w", err)
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *CredentialRepo) Delete(ctx context.Context, id string) error {
	const q = `DELETE FROM credentials WHERE id = $1`
	_, err := r.pool.Exec(ctx, q, id)
	if err != nil {
		return fmt.Errorf("storage: delete credential: %w", err)
	}
	return nil
}
