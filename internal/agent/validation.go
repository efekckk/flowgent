package agent

import (
	"encoding/json"
	"fmt"
	"strings"
)

type workflowDef struct {
	Name  string             `json:"name"`
	Nodes []map[string]any   `json:"nodes"`
	Edges []map[string]any   `json:"edges"`
}

// ValidateWorkflow returns a list of human-readable errors. An empty result
// means the workflow is acceptable to persist. The list is intended to be
// fed back to the chat provider verbatim when the validation feedback loop
// fires.
func ValidateWorkflow(raw []byte, knownTools map[string]struct{}) []string {
	var wf workflowDef
	if err := json.Unmarshal(raw, &wf); err != nil {
		return []string{fmt.Sprintf("workflow is not valid JSON: %v", err)}
	}
	var errs []string

	if wf.Name == "" {
		errs = append(errs, `field "name" is required and must be a non-empty string`)
	}
	if len(wf.Nodes) == 0 {
		errs = append(errs, `field "nodes" is required and must contain at least one node`)
	}

	idSet := make(map[string]struct{}, len(wf.Nodes))
	for i, n := range wf.Nodes {
		id, _ := n["id"].(string)
		tool, _ := n["tool"].(string)
		if id == "" {
			errs = append(errs, fmt.Sprintf("nodes[%d] missing \"id\"", i))
			continue
		}
		if _, dup := idSet[id]; dup {
			errs = append(errs, fmt.Sprintf("nodes[%d] duplicate id %q", i, id))
			continue
		}
		idSet[id] = struct{}{}
		if tool == "" {
			errs = append(errs, fmt.Sprintf("node %q missing \"tool\"", id))
			continue
		}
		if _, ok := knownTools[tool]; !ok {
			errs = append(errs, fmt.Sprintf("node %q references unknown tool %q", id, tool))
		}
	}

	// Edges: every from/to must reference a known node.
	for i, e := range wf.Edges {
		from, _ := e["from"].(string)
		to, _ := e["to"].(string)
		if from == "" || to == "" {
			errs = append(errs, fmt.Sprintf("edges[%d] missing from/to", i))
			continue
		}
		if _, ok := idSet[from]; !ok {
			errs = append(errs, fmt.Sprintf("edge %q -> %q: from references unknown node %q", from, to, from))
		}
		if _, ok := idSet[to]; !ok {
			errs = append(errs, fmt.Sprintf("edge %q -> %q: to references unknown node %q", from, to, to))
		}
	}

	if hasCycle(wf.Nodes, wf.Edges) {
		errs = append(errs, "graph contains a cycle (every workflow must be a DAG)")
	}

	return errs
}

// hasCycle performs a DFS from each node, detecting back-edges.
func hasCycle(nodes []map[string]any, edges []map[string]any) bool {
	adj := make(map[string][]string)
	for _, e := range edges {
		from, _ := e["from"].(string)
		to, _ := e["to"].(string)
		if from != "" && to != "" {
			adj[from] = append(adj[from], to)
		}
	}

	const (
		white = 0
		gray  = 1
		black = 2
	)
	color := make(map[string]int, len(nodes))
	for _, n := range nodes {
		id, _ := n["id"].(string)
		if id != "" {
			color[id] = white
		}
	}
	var dfs func(string) bool
	dfs = func(u string) bool {
		color[u] = gray
		for _, v := range adj[u] {
			if color[v] == gray {
				return true
			}
			if color[v] == white && dfs(v) {
				return true
			}
		}
		color[u] = black
		return false
	}
	for u, c := range color {
		if c == white && dfs(u) {
			return true
		}
	}
	return false
}

// FormatErrors renders the validation errors as a bullet list suitable for
// returning to the provider in a tool_result message.
func FormatErrors(errs []string) string {
	if len(errs) == 0 {
		return ""
	}
	var b strings.Builder
	for _, e := range errs {
		b.WriteString("- ")
		b.WriteString(e)
		b.WriteString("\n")
	}
	return b.String()
}
