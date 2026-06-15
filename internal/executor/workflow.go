package executor

// Workflow is the engine-facing form of a workflow definition. It is decoded
// from the workflow_versions.definition JSONB column and is treated as
// immutable during a run.
type Workflow struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Trigger struct {
		Type    string         `json:"type"`
		Payload map[string]any `json:"payload"`
	} `json:"trigger"`
	Nodes []Node `json:"nodes"`
	Edges []Edge `json:"edges"`
}

type Node struct {
	ID         string         `json:"id"`
	Tool       string         `json:"tool"`
	Params     map[string]any `json:"params"`
	Credential string         `json:"credential,omitempty"`
}

type Edge struct {
	From     string `json:"from"`
	FromPort string `json:"from_port"`
	To       string `json:"to"`
	ToPort   string `json:"to_port"`
}

func (w *Workflow) NodeByID(id string) (Node, bool) {
	for _, n := range w.Nodes {
		if n.ID == id {
			return n, true
		}
	}
	return Node{}, false
}

// EntryNodes returns nodes with no incoming edges — the engine seeds the
// run from this set.
func (w *Workflow) EntryNodes() []Node {
	hasIncoming := make(map[string]bool, len(w.Nodes))
	for _, e := range w.Edges {
		hasIncoming[e.To] = true
	}
	var out []Node
	for _, n := range w.Nodes {
		if !hasIncoming[n.ID] {
			out = append(out, n)
		}
	}
	return out
}
