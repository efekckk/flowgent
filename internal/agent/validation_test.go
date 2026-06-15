package agent

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestValidateWorkflow_ok(t *testing.T) {
	wf := []byte(`{
		"name": "demo",
		"nodes": [
			{"id":"n1","tool":"core.set","params":{"values":{}}}
		],
		"edges": []
	}`)
	known := map[string]struct{}{"core.set": {}}
	errs := ValidateWorkflow(wf, known)
	if len(errs) != 0 {
		t.Errorf("errs: %+v", errs)
	}
}

func TestValidateWorkflow_unknownTool(t *testing.T) {
	wf := []byte(`{
		"name": "demo",
		"nodes": [
			{"id":"n1","tool":"slack.nope","params":{}}
		],
		"edges": []
	}`)
	known := map[string]struct{}{"core.set": {}, "http.request": {}}
	errs := ValidateWorkflow(wf, known)
	if len(errs) == 0 || !contains(errs, "slack.nope") {
		t.Errorf("expected unknown tool error: %+v", errs)
	}
}

func TestValidateWorkflow_missingNodes(t *testing.T) {
	wf := []byte(`{"name":"demo","edges":[]}`)
	known := map[string]struct{}{}
	errs := ValidateWorkflow(wf, known)
	if len(errs) == 0 {
		t.Errorf("expected error for missing nodes")
	}
}

func TestValidateWorkflow_edgeReferencesUnknownNode(t *testing.T) {
	wf := []byte(`{
		"name": "demo",
		"nodes": [{"id":"n1","tool":"core.set","params":{}}],
		"edges": [{"from":"n1","from_port":"main","to":"n2","to_port":"main"}]
	}`)
	known := map[string]struct{}{"core.set": {}}
	errs := ValidateWorkflow(wf, known)
	if len(errs) == 0 || !contains(errs, "n2") {
		t.Errorf("expected edge target error: %+v", errs)
	}
}

func TestValidateWorkflow_cycleDetected(t *testing.T) {
	wf := []byte(`{
		"name": "demo",
		"nodes": [
			{"id":"a","tool":"core.set","params":{}},
			{"id":"b","tool":"core.set","params":{}}
		],
		"edges": [
			{"from":"a","from_port":"main","to":"b","to_port":"main"},
			{"from":"b","from_port":"main","to":"a","to_port":"main"}
		]
	}`)
	known := map[string]struct{}{"core.set": {}}
	errs := ValidateWorkflow(wf, known)
	if len(errs) == 0 || !contains(errs, "cycle") {
		t.Errorf("expected cycle: %+v", errs)
	}
}

func TestValidateWorkflow_invalidJSON(t *testing.T) {
	errs := ValidateWorkflow([]byte(`{not json`), map[string]struct{}{})
	if len(errs) == 0 {
		t.Errorf("expected JSON parse error")
	}
}

func TestFormatErrors_joinsWithBullets(t *testing.T) {
	formatted := FormatErrors([]string{"first", "second"})
	if !strings.Contains(formatted, "- first") || !strings.Contains(formatted, "- second") {
		t.Errorf("got %q", formatted)
	}
}

func contains(errs []string, sub string) bool {
	for _, e := range errs {
		if strings.Contains(e, sub) {
			return true
		}
	}
	return false
}

var _ = json.RawMessage{} // keep import live in case future tests need it
