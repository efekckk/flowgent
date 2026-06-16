// Package postgresquery implements "postgres.query" — opens a pgx connection
// per execute using the credential's DSN, runs the parameterised query, and
// returns the rows as an array of column → value maps. Connection is closed
// before return. Connection errors are NOT wrapped because the DSN (and the
// password it contains) is embedded in pgx error messages — leaking them
// into run records would be a credential disclosure. SQL placeholders are
// passed through to pgx as positional args, so user-supplied values cannot
// inject SQL.
package postgresquery

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/efekckk/flowgent/internal/executor"
	"github.com/efekckk/flowgent/internal/registry"
)

type Executor struct{}

func New() *Executor { return &Executor{} }

func (e *Executor) Execute(ctx context.Context, input map[string]any) (registry.ExecuteResult, error) {
	query, _ := input["query"].(string)
	if query == "" {
		return registry.ExecuteResult{}, fmt.Errorf("postgres.query: missing \"query\"")
	}
	cred, ok := input["__credential"].(map[string]any)
	if !ok {
		return registry.ExecuteResult{}, fmt.Errorf("postgres.query: missing credential")
	}
	dsn, _ := cred["dsn"].(string)
	if dsn == "" {
		return registry.ExecuteResult{}, fmt.Errorf("postgres.query: credential missing \"dsn\"")
	}

	var args []any
	if raw, ok := input["args"].([]any); ok {
		args = raw
	}

	connectCtx, cancelConnect := context.WithTimeout(ctx, 10*time.Second)
	defer cancelConnect()
	conn, err := pgx.Connect(connectCtx, dsn)
	if err != nil {
		// Do not wrap err with %w — pgx connect errors embed the DSN, which
		// contains the credential password.
		return registry.ExecuteResult{}, fmt.Errorf("postgres.query: connect failed")
	}
	defer conn.Close(context.Background())

	rows, err := conn.Query(ctx, query, args...)
	if err != nil {
		// Query errors do not embed the DSN — wrap with %w so the engine can
		// classify validation failures (syntax, undefined column, etc.).
		return registry.ExecuteResult{}, fmt.Errorf("postgres.query: %s: %w", err.Error(), executor.ErrValidation)
	}
	defer rows.Close()

	cols := rows.FieldDescriptions()
	colNames := make([]string, len(cols))
	for i, c := range cols {
		colNames[i] = string(c.Name)
	}
	out := []map[string]any{}
	for rows.Next() {
		vals, err := rows.Values()
		if err != nil {
			return registry.ExecuteResult{}, fmt.Errorf("postgres.query: scan failed: %w", executor.ErrValidation)
		}
		row := make(map[string]any, len(cols))
		for i, name := range colNames {
			row[name] = vals[i]
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return registry.ExecuteResult{}, fmt.Errorf("postgres.query: iterate failed: %w", executor.ErrValidation)
	}

	return registry.ExecuteResult{
		Output: map[string]any{"rows": out, "count": len(out)},
		Port:   "main",
	}, nil
}
