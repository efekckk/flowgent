// Package storage owns all Postgres access. Repositories embed the pgxpool
// and each one is responsible for exactly one table. Migrations live in the
// adjacent migrations/ subdirectory and are applied via golang-migrate. The
// adjacency is required: go:embed cannot traverse outside the package.
package storage

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres" // registers "postgres://" scheme
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

type Postgres struct {
	Pool *pgxpool.Pool
}

func Open(ctx context.Context, dsn string) (*Postgres, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("storage: pgxpool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("storage: ping: %w", err)
	}
	return &Postgres{Pool: pool}, nil
}

func (p *Postgres) Close() { p.Pool.Close() }

// Migrate applies all pending migrations embedded into the binary. Safe to
// call repeatedly; no-op when up-to-date.
func Migrate(dsn string) error {
	src, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("storage: iofs source: %w", err)
	}
	m, err := migrate.NewWithSourceInstance("iofs", src, normaliseScheme(dsn))
	if err != nil {
		return fmt.Errorf("storage: migrate instance: %w", err)
	}
	defer m.Close()
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("storage: migrate up: %w", err)
	}
	return nil
}

// normaliseScheme forces the "postgres://" prefix that the migrate driver
// expects, regardless of whether the caller passed "postgres://" or
// "postgresql://".
func normaliseScheme(dsn string) string {
	if strings.HasPrefix(dsn, "postgresql://") {
		return "postgres://" + strings.TrimPrefix(dsn, "postgresql://")
	}
	return dsn
}
