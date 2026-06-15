// Package corecode implements "core.code" — a sandboxed JavaScript transform
// node. The runtime exposes a strict whitelist of bindings; require, fetch,
// timers, eval, and Function are blocked by simply not being injected.
package corecode

import (
	"context"
	"fmt"
	"time"

	"github.com/dop251/goja"
)

const (
	defaultCPUTimeout = 1000 * time.Millisecond
	maxSnippetBytes   = 50 * 1024
)

// run compiles and executes the JS snippet against the provided bindings.
// All cancellation paths (ctx done, timeout) return an error.
func run(ctx context.Context, snippet string, bindings map[string]any) (any, error) {
	if len(snippet) > maxSnippetBytes {
		return nil, fmt.Errorf("core.code: snippet too large (max %d bytes)", maxSnippetBytes)
	}
	vm := goja.New()
	vm.SetMaxCallStackSize(100)
	for k, v := range bindings {
		_ = vm.Set(k, v)
	}

	timeout := defaultCPUTimeout
	tctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	interrupted := make(chan struct{})
	go func() {
		<-tctx.Done()
		vm.Interrupt("timeout")
		close(interrupted)
	}()

	wrapped := "(function(input, prev, $trigger, $now, $env) {" + snippet + "})(input, prev, $trigger, $now, $env)"
	value, err := vm.RunString(wrapped)
	cancel()
	<-interrupted
	if err != nil {
		return nil, fmt.Errorf("core.code: %w", err)
	}
	if value == nil || goja.IsUndefined(value) || goja.IsNull(value) {
		return map[string]any{}, nil
	}
	return value.Export(), nil
}
