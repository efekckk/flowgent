// Package storagetest spins up a throwaway Postgres container for tests.
// Start() is called once per package (TestMain); Fresh() returns a clean
// database isolated per test by name. Containers auto-purge on test exit.
package storagetest

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"

	"github.com/efekckk/flowgent/internal/storage"
)

var (
	pool     *dockertest.Pool
	resource *dockertest.Resource
	adminDSN string
)

func Start() error {
	var err error
	pool, err = dockertest.NewPool("")
	if err != nil {
		return fmt.Errorf("dockertest pool: %w", err)
	}
	if err := pool.Client.Ping(); err != nil {
		return fmt.Errorf("docker daemon: %w", err)
	}
	resource, err = pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "postgres",
		Tag:        "16-alpine",
		Env: []string{
			"POSTGRES_PASSWORD=secret",
			"POSTGRES_USER=postgres",
			"POSTGRES_DB=postgres",
			"listen_addresses='*'",
		},
	}, func(c *docker.HostConfig) {
		c.AutoRemove = true
		c.RestartPolicy = docker.RestartPolicy{Name: "no"}
	})
	if err != nil {
		return fmt.Errorf("run postgres: %w", err)
	}
	if err := resource.Expire(180); err != nil {
		log.Printf("expire warn: %v", err)
	}

	port := resource.GetPort("5432/tcp")
	adminDSN = fmt.Sprintf("postgres://postgres:secret@localhost:%s/postgres?sslmode=disable", port)

	pool.MaxWait = 90 * time.Second
	if err := pool.Retry(func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		p, err := pgxpool.New(ctx, adminDSN)
		if err != nil {
			return err
		}
		defer p.Close()
		return p.Ping(ctx)
	}); err != nil {
		return fmt.Errorf("postgres did not become ready: %w", err)
	}
	return nil
}

func Stop() {
	if pool != nil && resource != nil {
		_ = pool.Purge(resource)
	}
}

// Fresh creates a new database, applies all migrations, and returns its DSN.
// Database is dropped via t.Cleanup.
func Fresh(t *testing.T) string {
	t.Helper()
	dbName := "test_" + strings.ToLower(strings.ReplaceAll(t.Name(), "/", "_"))
	dbName = strings.ReplaceAll(dbName, ".", "_")
	if len(dbName) > 60 {
		dbName = dbName[:60]
	}

	ctx := context.Background()
	admin, err := pgxpool.New(ctx, adminDSN)
	if err != nil {
		t.Fatalf("admin pool: %v", err)
	}
	defer admin.Close()

	_, _ = admin.Exec(ctx, fmt.Sprintf(`DROP DATABASE IF EXISTS %q WITH (FORCE)`, dbName))
	if _, err := admin.Exec(ctx, fmt.Sprintf(`CREATE DATABASE %q`, dbName)); err != nil {
		t.Fatalf("create db: %v", err)
	}

	dsn := strings.Replace(adminDSN, "/postgres?", "/"+dbName+"?", 1)
	if err := storage.Migrate(dsn); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	t.Cleanup(func() {
		drop, err := pgxpool.New(context.Background(), adminDSN)
		if err != nil {
			return
		}
		defer drop.Close()
		_, _ = drop.Exec(context.Background(), fmt.Sprintf(`DROP DATABASE %q WITH (FORCE)`, dbName))
	})
	return dsn
}

func TestMain(m *testing.M) {
	if err := Start(); err != nil {
		log.Printf("storagetest: skipping (docker unavailable): %v", err)
		os.Exit(m.Run())
	}
	code := m.Run()
	Stop()
	os.Exit(code)
}
