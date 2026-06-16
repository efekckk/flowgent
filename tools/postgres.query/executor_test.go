package postgresquery

import (
	"context"
	"errors"
	"strings"
	"testing"

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
