package executor

import "github.com/efekckk/flowgent/internal/registry"

// Reexport so engine callers don't need to import both packages just to
// reference the executor's result type.
type ExecuteResult = registry.ExecuteResult
type Executor = registry.Executor
