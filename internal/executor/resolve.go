package executor

import (
	"fmt"

	"github.com/efekckk/flowgent/internal/expression"
)

// ResolveInputs walks the node's params map and renders any "{{ ... }}"
// expressions it finds in string values. Other types are passed through.
// Maps and slices are recursed into.
func ResolveInputs(params map[string]any, nodeID string, ctx expression.EvalContext, _ *RunState) (map[string]any, error) {
	out := make(map[string]any, len(params))
	for k, v := range params {
		rv, err := walk(v, ctx)
		if err != nil {
			return nil, fmt.Errorf("resolve %s.%s: %w", nodeID, k, err)
		}
		out[k] = rv
	}
	return out, nil
}

func walk(v any, ctx expression.EvalContext) (any, error) {
	switch t := v.(type) {
	case string:
		return expression.RenderValue(t, ctx)
	case map[string]any:
		out := make(map[string]any, len(t))
		for k, val := range t {
			rv, err := walk(val, ctx)
			if err != nil {
				return nil, err
			}
			out[k] = rv
		}
		return out, nil
	case []any:
		out := make([]any, len(t))
		for i, val := range t {
			rv, err := walk(val, ctx)
			if err != nil {
				return nil, err
			}
			out[i] = rv
		}
		return out, nil
	default:
		return v, nil
	}
}
