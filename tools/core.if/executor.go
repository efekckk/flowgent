// Package coreif implements "core.if" — a branching node that compares two
// values with an operator and routes execution to "true" or "false". The
// engine consumes the result's Port to decide which downstream edge fires.
package coreif

import (
	"context"
	"fmt"
	"strings"

	"github.com/efekckk/flowgent/internal/registry"
)

type Executor struct{}

func New() *Executor { return &Executor{} }

func (e *Executor) Execute(_ context.Context, input map[string]any) (registry.ExecuteResult, error) {
	left, ok := input["left"]
	if !ok {
		return registry.ExecuteResult{}, fmt.Errorf("core.if: missing \"left\"")
	}
	right, ok := input["right"]
	if !ok {
		return registry.ExecuteResult{}, fmt.Errorf("core.if: missing \"right\"")
	}
	opRaw, ok := input["op"].(string)
	if !ok {
		return registry.ExecuteResult{}, fmt.Errorf("core.if: \"op\" must be a string")
	}

	matched, err := compare(left, opRaw, right)
	if err != nil {
		return registry.ExecuteResult{}, fmt.Errorf("core.if: %w", err)
	}
	port := "false"
	if matched {
		port = "true"
	}
	return registry.ExecuteResult{
		Output: map[string]any{"matched": matched},
		Port:   port,
	}, nil
}

func compare(l any, op string, r any) (bool, error) {
	switch op {
	case "==":
		return equal(l, r), nil
	case "!=":
		return !equal(l, r), nil
	case ">":
		return numeric(l, r, func(a, b float64) bool { return a > b })
	case ">=":
		return numeric(l, r, func(a, b float64) bool { return a >= b })
	case "<":
		return numeric(l, r, func(a, b float64) bool { return a < b })
	case "<=":
		return numeric(l, r, func(a, b float64) bool { return a <= b })
	case "contains":
		ls, ok := l.(string)
		if !ok {
			return false, fmt.Errorf("contains: left must be string, got %T", l)
		}
		rs, ok := r.(string)
		if !ok {
			return false, fmt.Errorf("contains: right must be string, got %T", r)
		}
		return strings.Contains(ls, rs), nil
	default:
		return false, fmt.Errorf("unsupported op %q", op)
	}
}

func equal(a, b any) bool {
	if af, errA := toFloat(a); errA == nil {
		if bf, errB := toFloat(b); errB == nil {
			return af == bf
		}
	}
	return fmt.Sprint(a) == fmt.Sprint(b)
}

func numeric(l, r any, fn func(float64, float64) bool) (bool, error) {
	lf, err := toFloat(l)
	if err != nil {
		return false, fmt.Errorf("left not numeric: %w", err)
	}
	rf, err := toFloat(r)
	if err != nil {
		return false, fmt.Errorf("right not numeric: %w", err)
	}
	return fn(lf, rf), nil
}

func toFloat(v any) (float64, error) {
	switch n := v.(type) {
	case int:
		return float64(n), nil
	case int64:
		return float64(n), nil
	case float64:
		return n, nil
	case float32:
		return float64(n), nil
	default:
		return 0, fmt.Errorf("not numeric: %T", v)
	}
}
