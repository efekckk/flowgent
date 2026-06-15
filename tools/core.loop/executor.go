// Package coreloop implements "core.loop" — a control node that the engine
// recognises specially. The standalone tool here just validates inputs and
// returns metadata; the engine inspects core.loop nodes and runs the body
// once per item, threading $nodes.<loop_id>.current/index/total through.
package coreloop

import (
	"context"
	"fmt"

	"github.com/efekckk/flowgent/internal/registry"
)

type Executor struct{}

func New() *Executor { return &Executor{} }

func (e *Executor) Execute(_ context.Context, input map[string]any) (registry.ExecuteResult, error) {
	items, ok := input["items"].([]any)
	if !ok {
		return registry.ExecuteResult{}, fmt.Errorf("core.loop: \"items\" must be an array")
	}
	if _, ok := input["body"].([]any); !ok {
		return registry.ExecuteResult{}, fmt.Errorf("core.loop: \"body\" must be an array of node IDs")
	}
	return registry.ExecuteResult{
		Output: map[string]any{
			"total": len(items),
		},
		Port: "metadata",
	}, nil
}
