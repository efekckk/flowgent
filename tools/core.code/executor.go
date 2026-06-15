package corecode

import (
	"context"
	"fmt"
	"time"

	"github.com/efekckk/flowgent/internal/registry"
)

type Executor struct{}

func New() *Executor { return &Executor{} }

func (e *Executor) Execute(ctx context.Context, input map[string]any) (registry.ExecuteResult, error) {
	code, ok := input["code"].(string)
	if !ok {
		return registry.ExecuteResult{}, fmt.Errorf("core.code: missing \"code\" input")
	}
	bindings := map[string]any{
		"input":    input,
		"prev":     input["__prev"],
		"$now":     time.Now().UTC().Format(time.RFC3339),
		"$trigger": nil,
		"$env":     nil,
	}
	out, err := run(ctx, code, bindings)
	if err != nil {
		return registry.ExecuteResult{}, err
	}
	asMap, ok := out.(map[string]any)
	if !ok {
		asMap = map[string]any{"value": out}
	}
	return registry.ExecuteResult{Output: asMap, Port: "main"}, nil
}
