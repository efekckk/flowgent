// Package coreset implements "core.set" — a transform node whose output is
// exactly its input "values" map. Use it to materialise computed fields
// (via expressions in the values) for consumption by downstream nodes.
package coreset

import (
	"context"
	"fmt"

	"github.com/efekckk/flowgent/internal/registry"
)

type Executor struct{}

func New() *Executor { return &Executor{} }

func (e *Executor) Execute(_ context.Context, input map[string]any) (registry.ExecuteResult, error) {
	raw, ok := input["values"]
	if !ok {
		return registry.ExecuteResult{}, fmt.Errorf("core.set: missing input \"values\"")
	}
	values, ok := raw.(map[string]any)
	if !ok {
		return registry.ExecuteResult{}, fmt.Errorf("core.set: \"values\" must be an object, got %T", raw)
	}
	return registry.ExecuteResult{Output: values, Port: "main"}, nil
}
