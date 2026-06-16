package postgresquery

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/efekckk/flowgent/internal/executor"
	"github.com/efekckk/flowgent/internal/storage/storagetest"
)

func TestExecute_selectsLiteralRow(t *testing.T) {
	dsn := storagetest.Fresh(t)
	e := New()
	res, err := e.Execute(context.Background(), map[string]any{
		"query":        "SELECT 'hello' AS greeting, 42 AS answer",
		"__credential": map[string]any{"dsn": dsn, "__type": "postgres"},
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	rows, _ := res.Output["rows"].([]map[string]any)
	if len(rows) != 1 {
		t.Fatalf("rows: %+v", res.Output)
	}
	if rows[0]["greeting"] != "hello" {
		t.Errorf("greeting: %v", rows[0]["greeting"])
	}
	if cnt, _ := res.Output["count"].(int); cnt != 1 {
		t.Errorf("count: %v (%T)", res.Output["count"], res.Output["count"])
	}
}

func TestExecute_parameterisedQuery(t *testing.T) {
	dsn := storagetest.Fresh(t)
	e := New()
	res, err := e.Execute(context.Background(), map[string]any{
		"query":        "SELECT $1::int AS n",
		"args":         []any{7},
		"__credential": map[string]any{"dsn": dsn, "__type": "postgres"},
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	rows, _ := res.Output["rows"].([]map[string]any)
	if len(rows) != 1 {
		t.Fatalf("rows: %+v", rows)
	}
}

func TestExecute_multipleRows(t *testing.T) {
	dsn := storagetest.Fresh(t)
	e := New()
	res, err := e.Execute(context.Background(), map[string]any{
		"query":        "SELECT generate_series(1, 3) AS n",
		"__credential": map[string]any{"dsn": dsn, "__type": "postgres"},
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	rows, _ := res.Output["rows"].([]map[string]any)
	if len(rows) != 3 {
		t.Fatalf("expected 3 rows, got %d: %+v", len(rows), rows)
	}
}

func TestExecute_invalidSQLIsValidationError(t *testing.T) {
	dsn := storagetest.Fresh(t)
	e := New()
	_, err := e.Execute(context.Background(), map[string]any{
		"query":        "SELEKT bogus",
		"__credential": map[string]any{"dsn": dsn, "__type": "postgres"},
	})
	if err == nil {
		t.Fatalf("expected error")
	}
	if !errors.Is(err, executor.ErrValidation) {
		t.Fatalf("expected ErrValidation, got %v", err)
	}
}

func TestExecute_missingQueryIsError(t *testing.T) {
	e := New()
	_, err := e.Execute(context.Background(), map[string]any{
		"__credential": map[string]any{"dsn": "postgres://x", "__type": "postgres"},
	})
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestExecute_missingCredentialIsError(t *testing.T) {
	e := New()
	_, err := e.Execute(context.Background(), map[string]any{"query": "SELECT 1"})
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestExecute_missingDSNIsError(t *testing.T) {
	e := New()
	_, err := e.Execute(context.Background(), map[string]any{
		"query":        "SELECT 1",
		"__credential": map[string]any{"__type": "postgres"},
	})
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestExecute_connectFailureDoesNotLeakDSN(t *testing.T) {
	// Point at a DSN with a fake password to a non-existent host.
	e := New()
	secretDSN := "postgres://user:supersecretpw@127.0.0.1:1/none?sslmode=disable&connect_timeout=1"
	_, err := e.Execute(context.Background(), map[string]any{
		"query":        "SELECT 1",
		"__credential": map[string]any{"dsn": secretDSN, "__type": "postgres"},
	})
	if err == nil {
		t.Fatalf("expected connect failure")
	}
	if strings.Contains(err.Error(), "supersecretpw") {
		t.Errorf("DSN password leaked into error: %v", err)
	}
	if strings.Contains(err.Error(), secretDSN) {
		t.Errorf("full DSN leaked into error: %v", err)
	}
}

func TestExecute_emptyResultSet(t *testing.T) {
	dsn := storagetest.Fresh(t)
	e := New()
	res, err := e.Execute(context.Background(), map[string]any{
		"query":        "SELECT 1 AS n WHERE false",
		"__credential": map[string]any{"dsn": dsn, "__type": "postgres"},
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	rows, _ := res.Output["rows"].([]map[string]any)
	if rows == nil {
		t.Errorf("rows should be empty slice, not nil")
	}
	if len(rows) != 0 {
		t.Errorf("rows: %+v", rows)
	}
	if cnt, _ := res.Output["count"].(int); cnt != 0 {
		t.Errorf("count: %v", res.Output["count"])
	}
}

func TestExecute_connectErrorIsTransient(t *testing.T) {
	e := New()
	_, err := e.Execute(context.Background(), map[string]any{
		"query": "SELECT 1",
		"__credential": map[string]any{
			"dsn":    "postgres://user:pw@127.0.0.1:1/none?sslmode=disable&connect_timeout=1",
			"__type": "postgres",
		},
	})
	if err == nil {
		t.Fatalf("expected error")
	}
	if !errors.Is(err, executor.ErrTransient5xx) {
		t.Fatalf("expected ErrTransient5xx, got %v", err)
	}
}

func TestExecute_syntaxErrorIsValidation(t *testing.T) {
	dsn := storagetest.Fresh(t)
	e := New()
	_, err := e.Execute(context.Background(), map[string]any{
		"query":        "SELECT * FROM nonexistent_table_xyz",
		"__credential": map[string]any{"dsn": dsn, "__type": "postgres"},
	})
	if err == nil || !errors.Is(err, executor.ErrValidation) {
		t.Fatalf("expected ErrValidation, got %v", err)
	}
}

func TestExecute_constraintErrorIsValidation(t *testing.T) {
	dsn := storagetest.Fresh(t)
	e := New()
	// Use a built-in constraint: dividing by zero -> 22012, class 22 (data exception) -> validation
	_, err := e.Execute(context.Background(), map[string]any{
		"query":        "SELECT 1/0",
		"__credential": map[string]any{"dsn": dsn, "__type": "postgres"},
	})
	if err == nil || !errors.Is(err, executor.ErrValidation) {
		t.Fatalf("expected ErrValidation, got %v", err)
	}
}

func TestExecute_queryTimeoutEnforced(t *testing.T) {
	// 30s default is fine for tests; this test simply verifies a long-running
	// query CAN be cancelled. We use a tiny ctx deadline instead of waiting 30s.
	dsn := storagetest.Fresh(t)
	e := New()
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	_, err := e.Execute(ctx, map[string]any{
		"query":        "SELECT pg_sleep(5)",
		"__credential": map[string]any{"dsn": dsn, "__type": "postgres"},
	})
	if err == nil {
		t.Fatalf("expected timeout error")
	}
	if !errors.Is(err, executor.ErrTransient5xx) {
		t.Fatalf("expected ErrTransient5xx (canceled query is transient), got %v", err)
	}
}
