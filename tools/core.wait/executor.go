// Package corewait implements the "core.wait" tool, which pauses workflow
// execution for a fixed number of milliseconds. It honours ctx cancellation
// so callers can abort a long sleep without leaking a goroutine.
package corewait

import (
	"context"
	"fmt"
	"time"

	"github.com/efekckk/flowgent/internal/registry"
)

type Executor struct{}

func New() *Executor { return &Executor{} }

func (e *Executor) Execute(ctx context.Context, input map[string]any) (registry.ExecuteResult, error) {
	raw, ok := input["ms"]
	if !ok {
		return registry.ExecuteResult{}, fmt.Errorf("core.wait: missing input \"ms\"")
	}
	ms, err := toInt64(raw)
	if err != nil {
		return registry.ExecuteResult{}, fmt.Errorf("core.wait: %w", err)
	}
	if ms < 0 {
		return registry.ExecuteResult{}, fmt.Errorf("core.wait: ms must be non-negative")
	}
	timer := time.NewTimer(time.Duration(ms) * time.Millisecond)
	defer timer.Stop()
	select {
	case <-timer.C:
		return registry.ExecuteResult{
			Output: map[string]any{"waited_ms": ms},
			Port:   "main",
		}, nil
	case <-ctx.Done():
		return registry.ExecuteResult{}, ctx.Err()
	}
}

func toInt64(v any) (int64, error) {
	switch n := v.(type) {
	case int:
		return int64(n), nil
	case int64:
		return n, nil
	case float64:
		return int64(n), nil
	case float32:
		return int64(n), nil
	default:
		return 0, fmt.Errorf("expected integer, got %T", v)
	}
}
