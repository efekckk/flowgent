package storage

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ChatThread struct {
	ID         string
	WorkflowID string
	UserID     string
	CreatedAt  time.Time
}

type ChatThreadRepo struct {
	pool *pgxpool.Pool
}

func NewChatThreadRepo(pool *pgxpool.Pool) *ChatThreadRepo { return &ChatThreadRepo{pool: pool} }

func (r *ChatThreadRepo) Insert(ctx context.Context, t ChatThread) error {
	const q = `INSERT INTO chat_threads (id, workflow_id, user_id) VALUES ($1, $2, $3)`
	if _, err := r.pool.Exec(ctx, q, t.ID, t.WorkflowID, t.UserID); err != nil {
		return fmt.Errorf("storage: insert chat thread: %w", err)
	}
	return nil
}

func (r *ChatThreadRepo) GetByWorkflowAndUser(ctx context.Context, workflowID, userID string) (ChatThread, error) {
	const q = `
		SELECT id, workflow_id, user_id, created_at
		FROM chat_threads WHERE workflow_id = $1 AND user_id = $2
		ORDER BY created_at DESC LIMIT 1
	`
	var t ChatThread
	err := r.pool.QueryRow(ctx, q, workflowID, userID).Scan(&t.ID, &t.WorkflowID, &t.UserID, &t.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return ChatThread{}, ErrNotFound
	}
	if err != nil {
		return ChatThread{}, fmt.Errorf("storage: get chat thread: %w", err)
	}
	return t, nil
}
