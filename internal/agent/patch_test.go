package agent

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestApplyPatches_replaceName(t *testing.T) {
	original := []byte(`{"name":"old","nodes":[],"edges":[]}`)
	patches := []byte(`[{"op":"replace","path":"/name","value":"new"}]`)
	result, err := ApplyPatches(original, patches)
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if !strings.Contains(string(result), `"name":"new"`) {
		t.Errorf("got %s", string(result))
	}
}

func TestApplyPatches_addNode(t *testing.T) {
	original := []byte(`{"name":"x","nodes":[],"edges":[]}`)
	patches := []byte(`[{
		"op":"add","path":"/nodes/-",
		"value":{"id":"n1","tool":"core.set","params":{"values":{}}}
	}]`)
	result, err := ApplyPatches(original, patches)
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	var parsed map[string]any
	_ = json.Unmarshal(result, &parsed)
	nodes, _ := parsed["nodes"].([]any)
	if len(nodes) != 1 {
		t.Errorf("nodes: %+v", nodes)
	}
}

func TestApplyPatches_invalidJSONReturnsError(t *testing.T) {
	_, err := ApplyPatches([]byte(`{bad`), []byte(`[]`))
	if err == nil {
		t.Fatalf("expected error on invalid original")
	}
}

func TestApplyPatches_invalidPatchOpReturnsError(t *testing.T) {
	_, err := ApplyPatches([]byte(`{}`), []byte(`[{"op":"bogus","path":"/x"}]`))
	if err == nil {
		t.Fatalf("expected error on bogus op")
	}
}
