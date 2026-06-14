package storage

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Session struct {
	TokenHash  []byte
	UserID     string
	CreatedAt  time.Time
	ExpiresAt  time.Time
	LastUsedAt time.Time
	IP         string
	UserAgent  string
}

type SessionRepo struct {
	pool *pgxpool.Pool
}

func NewSessionRepo(pool *pgxpool.Pool) *SessionRepo { return &SessionRepo{pool: pool} }

func (r *SessionRepo) Insert(ctx context.Context, s Session) error {
	const q = `
		INSERT INTO sessions (token_hash, user_id, expires_at, ip, user_agent)
		VALUES ($1, $2, $3, NULLIF($4,'')::inet, NULLIF($5,''))
	`
	if _, err := r.pool.Exec(ctx, q, s.TokenHash, s.UserID, s.ExpiresAt, s.IP, s.UserAgent); err != nil {
		return fmt.Errorf("storage: insert session: %w", err)
	}
	return nil
}

// FindByTokenHash returns the session only when not expired. Rows past
// expires_at are treated as missing so callers don't have to repeat the check.
func (r *SessionRepo) FindByTokenHash(ctx context.Context, hash []byte) (Session, error) {
	const q = `
		SELECT token_hash, user_id, created_at, expires_at, last_used_at,
		       COALESCE(host(ip), ''), COALESCE(user_agent, '')
		FROM sessions
		WHERE token_hash = $1 AND expires_at > now()
	`
	var s Session
	err := r.pool.QueryRow(ctx, q, hash).Scan(
		&s.TokenHash, &s.UserID, &s.CreatedAt, &s.ExpiresAt, &s.LastUsedAt, &s.IP, &s.UserAgent,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Session{}, ErrNotFound
	}
	if err != nil {
		return Session{}, fmt.Errorf("storage: find session: %w", err)
	}
	return s, nil
}

func (r *SessionRepo) Touch(ctx context.Context, hash []byte) error {
	const q = `UPDATE sessions SET last_used_at = now() WHERE token_hash = $1`
	_, err := r.pool.Exec(ctx, q, hash)
	return err
}

func (r *SessionRepo) Delete(ctx context.Context, hash []byte) error {
	const q = `DELETE FROM sessions WHERE token_hash = $1`
	_, err := r.pool.Exec(ctx, q, hash)
	return err
}

func (r *SessionRepo) PurgeExpired(ctx context.Context) (int64, error) {
	const q = `DELETE FROM sessions WHERE expires_at <= now()`
	tag, err := r.pool.Exec(ctx, q)
	if err != nil {
		return 0, fmt.Errorf("storage: purge sessions: %w", err)
	}
	return tag.RowsAffected(), nil
}
