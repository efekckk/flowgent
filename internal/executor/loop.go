package executor

import (
	"context"
	"fmt"

	"github.com/efekckk/flowgent/internal/expression"
)

// IsLoopNode reports whether a node is a core.loop node.
func IsLoopNode(n *Node) bool { return n.Tool == "core.loop" }

// runLoop iterates over the loop's items, executing each body node once
// per iteration. Updates state.NodeOutputs[loopID] with current/index/total
// before each body iteration, and with iterations/items after all done.
// Returns the port the loop "fires" — always "done".
func (e *Engine) runLoop(ctx context.Context, wf *Workflow, node *Node, state *RunState, opts RunOptions) (string, error) {
	state.SetStatus(node.ID, "running")
	evalCtx := expression.EvalContext{
		Trigger: opts.TriggerPayload,
		Nodes:   state.LatestOutputsMap(),
	}
	resolved, err := ResolveInputs(node.Params, node.ID, evalCtx, state)
	if err != nil {
		state.SetStatus(node.ID, "failed")
		state.SetRunStatus("failed", err.Error())
		return "", err
	}
	state.SetInput(node.ID, resolved)

	items, ok := resolved["items"].([]any)
	if !ok {
		state.SetStatus(node.ID, "failed")
		err := fmt.Errorf("core.loop %s: \"items\" must be an array", node.ID)
		state.SetRunStatus("failed", err.Error())
		return "", err
	}
	bodyRaw, _ := resolved["body"].([]any)
	bodyIDs := make([]string, 0, len(bodyRaw))
	for _, b := range bodyRaw {
		if s, ok := b.(string); ok {
			bodyIDs = append(bodyIDs, s)
		}
	}

	for i, item := range items {
		// Surface iteration metadata via $nodes.<loop_id>
		state.RecordOutput(node.ID, i, map[string]any{
			"current": item,
			"index":   i,
			"total":   len(items),
		}, "metadata")

		for _, bid := range bodyIDs {
			bnode, ok := wf.NodeByID(bid)
			if !ok {
				err := fmt.Errorf("core.loop %s: unknown body node %q", node.ID, bid)
				state.SetRunStatus("failed", err.Error())
				return "", err
			}
			state.SetStatus(bnode.ID, "pending")
			if _, err := e.executeNode(ctx, wf, &bnode, state, opts); err != nil {
				return "", err
			}
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			default:
			}
		}
	}

	// Final loop output — overwrites the per-iteration metadata
	state.RecordOutput(node.ID, len(items), map[string]any{
		"iterations": len(items),
		"total":      len(items),
		"items":      items,
	}, "done")
	state.SetStatus(node.ID, "succeeded")
	return "done", nil
}
