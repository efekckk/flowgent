package agent

import (
	"encoding/json"
	"fmt"

	jsonpatch "github.com/evanphx/json-patch/v5"
)

// ApplyPatches applies an RFC 6902 JSON-Patch to a workflow definition and
// returns the modified bytes. The patches arrive as the raw "patches" field
// of the chat provider's edit_workflow tool call.
func ApplyPatches(original, patchOps []byte) ([]byte, error) {
	patch, err := jsonpatch.DecodePatch(patchOps)
	if err != nil {
		return nil, fmt.Errorf("agent: decode patch: %w", err)
	}
	out, err := patch.Apply(original)
	if err != nil {
		return nil, fmt.Errorf("agent: apply patch: %w", err)
	}
	// Validate that the result is still valid JSON
	var probe map[string]any
	if err := json.Unmarshal(out, &probe); err != nil {
		return nil, fmt.Errorf("agent: patched result not JSON: %w", err)
	}
	return out, nil
}
