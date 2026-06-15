// Package coremerge implements "core.merge" — a control node that combines
// outputs from multiple upstream branches into a single output. The engine
// is responsible for collecting upstream outputs into `inputs`; this tool
// only performs the requested combination.
package coremerge

import (
	"context"
	"fmt"

	"github.com/efekckk/flowgent/internal/registry"
)

type Executor struct{}

func New() *Executor { return &Executor{} }

func (e *Executor) Execute(_ context.Context, input map[string]any) (registry.ExecuteResult, error) {
	mode, _ := input["mode"].(string)
	rawInputs, ok := input["inputs"].([]any)
	if !ok {
		return registry.ExecuteResult{}, fmt.Errorf("core.merge: \"inputs\" must be an array")
	}

	switch mode {
	case "append":
		return registry.ExecuteResult{
			Output: map[string]any{"items": rawInputs},
			Port:   "main",
		}, nil
	case "merge_objects":
		merged := make(map[string]any)
		for _, item := range rawInputs {
			obj, ok := item.(map[string]any)
			if !ok {
				continue
			}
			for k, v := range obj {
				merged[k] = v
			}
		}
		return registry.ExecuteResult{Output: merged, Port: "main"}, nil
	case "first_to_complete":
		for _, item := range rawInputs {
			if item == nil {
				continue
			}
			obj, ok := item.(map[string]any)
			if !ok {
				continue
			}
			return registry.ExecuteResult{Output: obj, Port: "main"}, nil
		}
		return registry.ExecuteResult{Output: map[string]any{}, Port: "main"}, nil
	default:
		return registry.ExecuteResult{}, fmt.Errorf("core.merge: unknown mode %q", mode)
	}
}
