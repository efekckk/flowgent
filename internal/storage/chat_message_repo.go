package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type ChatMessage struct {
	ID          string
	ThreadID    string
	Seq         int64
	Role        string
	Content     string
	ToolCalls   json.RawMessage
	ToolResults json.RawMessage
	Model       string
	Usage       json.RawMessage
	CreatedAt   time.Time
}

type ChatMessageRepo struct {
	pool *pgxpool.Pool
}

func NewChatMessageRepo(pool *pgxpool.Pool) *ChatMessageRepo { return &ChatMessageRepo{pool: pool} }

func (r *ChatMessageRepo) Insert(ctx context.Context, m ChatMessage) error {
	const q = `
		INSERT INTO chat_messages (id, thread_id, role, content, tool_calls, tool_results, model, usage)
		VALUES ($1, $2, $3, NULLIF($4, ''), NULLIF($5::jsonb, 'null'::jsonb),
		        NULLIF($6::jsonb, 'null'::jsonb), NULLIF($7, ''), NULLIF($8::jsonb, 'null'::jsonb))
	`
	_, err := r.pool.Exec(ctx, q,
		m.ID, m.ThreadID, m.Role, m.Content,
		string(jsonOrNull(m.ToolCalls)), string(jsonOrNull(m.ToolResults)),
		m.Model, string(jsonOrNull(m.Usage)),
	)
	if err != nil {
		return fmt.Errorf("storage: insert chat message: %w", err)
	}
	return nil
}

func jsonOrNull(b json.RawMessage) json.RawMessage {
	if len(b) == 0 {
		return json.RawMessage("null")
	}
	return b
}

func (r *ChatMessageRepo) ListByThread(ctx context.Context, threadID string, limit int) ([]ChatMessage, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	const q = `
		SELECT id, thread_id, seq, role, COALESCE(content, ''),
		       COALESCE(tool_calls, 'null'::jsonb), COALESCE(tool_results, 'null'::jsonb),
		       COALESCE(model, ''), COALESCE(usage, 'null'::jsonb), created_at
		FROM chat_messages WHERE thread_id = $1
		ORDER BY seq ASC LIMIT $2
	`
	rows, err := r.pool.Query(ctx, q, threadID, limit)
	if err != nil {
		return nil, fmt.Errorf("storage: list chat messages: %w", err)
	}
	defer rows.Close()
	var out []ChatMessage
	for rows.Next() {
		var m ChatMessage
		if err := rows.Scan(&m.ID, &m.ThreadID, &m.Seq, &m.Role, &m.Content,
			&m.ToolCalls, &m.ToolResults, &m.Model, &m.Usage, &m.CreatedAt); err != nil {
			return nil, fmt.Errorf("storage: scan chat message: %w", err)
		}
		out = append(out, m)
	}
	return out, rows.Err()
}
