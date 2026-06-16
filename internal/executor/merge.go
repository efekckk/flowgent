package executor

// MergeMagicKey is the placeholder in merge node params that the engine
// substitutes with the array of upstream outputs before dispatching.
const MergeMagicKey = "$merge.upstream"

// IsMergeNode reports whether a node is a core.merge node.
func IsMergeNode(n *Node) bool { return n.Tool == "core.merge" }

// UpstreamNodes returns IDs of nodes whose edges point into nodeID.
func UpstreamNodes(wf *Workflow, nodeID string) []string {
	var out []string
	for _, e := range wf.Edges {
		if e.To == nodeID {
			out = append(out, e.From)
		}
	}
	return out
}

// AllUpstreamSucceeded returns true if every upstream of nodeID is in the
// "succeeded" status. Used to gate merge nodes.
func AllUpstreamSucceeded(wf *Workflow, nodeID string, state *RunState) bool {
	ups := UpstreamNodes(wf, nodeID)
	if len(ups) == 0 {
		return true
	}
	for _, up := range ups {
		if state.Status(up) != "succeeded" {
			return false
		}
	}
	return true
}

// pendingUpstreams returns the ids of upstreams that are not yet in the
// "succeeded" status. Used to populate merge-wait log events so a viewer can
// see which branch is holding the merge.
func pendingUpstreams(wf *Workflow, nodeID string, state *RunState) []string {
	ups := UpstreamNodes(wf, nodeID)
	out := make([]string, 0, len(ups))
	for _, up := range ups {
		if state.Status(up) != "succeeded" {
			out = append(out, up)
		}
	}
	return out
}

// CollectUpstreamOutputs gathers the latest outputs of every upstream into
// the order they appear in wf.Edges. Used to fill in merge inputs.
func CollectUpstreamOutputs(wf *Workflow, nodeID string, state *RunState) []any {
	ups := UpstreamNodes(wf, nodeID)
	out := make([]any, 0, len(ups))
	for _, up := range ups {
		out = append(out, state.LatestOutput(up))
	}
	return out
}
