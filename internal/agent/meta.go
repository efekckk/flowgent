// Package agent runs the workflow generation loop. Meta-tools here are
// exposed to the chat provider via the tool-call interface; they are
// distinct from workflow tools (core.set, http.request, etc.) which the
// engine executes when the workflow runs.
package agent

import (
	"encoding/json"

	"github.com/efekckk/flowgent/internal/provider"
)

// MetaTools returns the tool palette exposed during workflow generation
// and editing. M4 ships propose_workflow + edit_workflow.
func MetaTools() []provider.ToolDefinition {
	return []provider.ToolDefinition{
		{
			Name: "propose_workflow",
			Description: "Submit a new workflow DAG for the user to review. " +
				"Use this when the user asks for a workflow to be built from scratch.",
			InputSchema: json.RawMessage(proposeWorkflowSchema),
		},
		{
			Name: "edit_workflow",
			Description: "Modify the current workflow with JSON-Patch (RFC 6902) operations. " +
				"Use this when the user wants to change an existing workflow.",
			InputSchema: json.RawMessage(editWorkflowSchema),
		},
	}
}

const proposeWorkflowSchema = `{
  "type": "object",
  "required": ["name", "nodes", "edges"],
  "properties": {
    "name":  { "type": "string", "description": "Human-readable workflow name." },
    "nodes": {
      "type": "array",
      "items": {
        "type": "object",
        "required": ["id", "tool", "params"],
        "properties": {
          "id":    { "type": "string", "description": "Unique within this workflow." },
          "tool":  { "type": "string", "description": "Tool slug, e.g. \"http.request\" or \"core.set\"." },
          "params": { "type": "object", "description": "Tool input parameters; may contain {{ $trigger.* }} or {{ $nodes.<id>.* }} expressions." },
          "credential": { "type": "string", "description": "Optional credential ID reference." }
        }
      }
    },
    "edges": {
      "type": "array",
      "items": {
        "type": "object",
        "required": ["from", "from_port", "to", "to_port"],
        "properties": {
          "from":      { "type": "string" },
          "from_port": { "type": "string", "description": "Output port name on the source node." },
          "to":        { "type": "string" },
          "to_port":   { "type": "string", "description": "Input port name on the target node." }
        }
      }
    }
  }
}`

const editWorkflowSchema = `{
  "type": "object",
  "required": ["patches"],
  "properties": {
    "patches": {
      "type": "array",
      "description": "RFC 6902 JSON-Patch operations applied to the current workflow definition.",
      "items": {
        "type": "object",
        "required": ["op", "path"],
        "properties": {
          "op":    { "type": "string", "enum": ["add", "remove", "replace", "move", "copy", "test"] },
          "path":  { "type": "string" },
          "value": {},
          "from":  { "type": "string" }
        }
      }
    }
  }
}`
