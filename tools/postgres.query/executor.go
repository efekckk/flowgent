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
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

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
		// Do not surface err — pgx Connect errors embed the DSN (with password).
		return registry.ExecuteResult{}, errors.Join(
			fmt.Errorf("postgres.query: connect failed"),
			executor.ErrTransient5xx,
		)
	}
	defer conn.Close(context.Background())

	queryCtx, cancelQuery := context.WithTimeout(ctx, 30*time.Second)
	defer cancelQuery()
	rows, err := conn.Query(queryCtx, query, args...)
	if err != nil {
		// Query errors do not embed the DSN — wrap with %w so the engine can
		// classify validation vs. transient failures via SQLSTATE class.
		return registry.ExecuteResult{}, fmt.Errorf("postgres.query: %s: %w", err.Error(), classifyPgError(err))
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
			return registry.ExecuteResult{}, fmt.Errorf("postgres.query: scan failed: %s: %w", err.Error(), classifyPgError(err))
		}
		row := make(map[string]any, len(cols))
		for i, name := range colNames {
			row[name] = vals[i]
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return registry.ExecuteResult{}, fmt.Errorf("postgres.query: iterate failed: %s: %w", err.Error(), classifyPgError(err))
	}

	return registry.ExecuteResult{
		Output: map[string]any{"rows": out, "count": len(out)},
		Port:   "main",
	}, nil
}

// classifyPgError maps a pgx error to the engine's retry sentinels via
// SQLSTATE class:
//   08 — connection_exception  -> transient
//   40 — transaction_rollback / serialization_failure -> transient
//   57 — operator_intervention (admin_shutdown, query_canceled) -> transient
//   42 — syntax_error_or_access_rule_violation -> validation
//   53 — insufficient_resources -> transient
//   23 — integrity_constraint_violation -> validation
// Anything else (or non-PgError, including raw network errors during
// row iteration) is treated as transient — Postgres tools should default
// to retryable so transient blips don't permanently fail a workflow run.
func classifyPgError(err error) error {
	var pgerr *pgconn.PgError
	if errors.As(err, &pgerr) && len(pgerr.Code) >= 2 {
		switch pgerr.Code[:2] {
		case "42", "23", "22": // syntax, integrity, data exception -> non-retryable
			return executor.ErrValidation
		}
	}
	return executor.ErrTransient5xx
}
