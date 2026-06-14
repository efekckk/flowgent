package storage

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestOpen_pingsSuccessfully(t *testing.T) {
	dsn := os.Getenv("FLOWGENT_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("FLOWGENT_TEST_DATABASE_URL not set; run via dockertest harness in Task 5")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pg, err := Open(ctx, dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer pg.Close()

	if err := pg.Pool.Ping(ctx); err != nil {
		t.Fatalf("ping: %v", err)
	}
}
