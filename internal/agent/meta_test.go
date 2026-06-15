package agent

import (
	"strings"
	"testing"
)

func TestMetaTools_proposeWorkflowIsListed(t *testing.T) {
	tools := MetaTools()
	found := false
	for _, td := range tools {
		if td.Name == "propose_workflow" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("propose_workflow not in MetaTools: %+v", tools)
	}
}

func TestMetaTools_editWorkflowSchemaMentionsPatch(t *testing.T) {
	tools := MetaTools()
	for _, td := range tools {
		if td.Name == "edit_workflow" {
			if !strings.Contains(string(td.InputSchema), "patches") {
				t.Errorf("schema missing patches: %s", string(td.InputSchema))
			}
			return
		}
	}
	t.Errorf("edit_workflow not found")
}
