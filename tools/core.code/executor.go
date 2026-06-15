package corecode

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/efekckk/flowgent/internal/registry"
)

const maxOutputBytes = 1024 * 1024

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
		"$trigger": nil,
		"$now":     time.Now().UTC().Format(time.RFC3339),
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
	jsonBytes, err := json.Marshal(asMap)
	if err != nil {
		return registry.ExecuteResult{}, fmt.Errorf("core.code: output is not JSON-serialisable: %w", err)
	}
	if len(jsonBytes) > maxOutputBytes {
		return registry.ExecuteResult{}, fmt.Errorf("core.code: output exceeds %d bytes", maxOutputBytes)
	}
	return registry.ExecuteResult{Output: asMap, Port: "main"}, nil
}
