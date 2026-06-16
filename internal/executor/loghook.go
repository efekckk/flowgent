package executor

import "context"

// LogSink receives structured log events emitted by the engine during a run.
// Implementations typically persist to storage.RunLogRepo and/or fan out to
// SSE subscribers. Errors are intentionally ignored by the engine — a sink
// failure must never crash a workflow.
//
// Messages are formed from the tool slug, node id, and classified errors
// only; raw inputs (which may contain a resolved "__credential") and raw
// outputs (which may contain user content) are never embedded.
type LogSink interface {
	Append(ctx context.Context, runID, nodeID, level, message string)
}

type noopSink struct{}

func (noopSink) Append(_ context.Context, _, _, _, _ string) {}

// WithLogSink installs a LogSink for the engine. Passing nil restores the
// noop sink so callers can disable observability without panicking.
func WithLogSink(s LogSink) Option {
	return func(e *Engine) {
		if s == nil {
			s = noopSink{}
		}
		e.logSink = s
	}
}
