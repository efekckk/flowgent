package executor

import (
	"reflect"
	"testing"

	"github.com/efekckk/flowgent/internal/expression"
)

func TestResolveInputs_literal(t *testing.T) {
	state := newRunState()
	got, err := ResolveInputs(map[string]any{"k": "v"}, "n1", expression.EvalContext{}, &state)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if got["k"] != "v" {
		t.Errorf("got %+v", got)
	}
}

func TestResolveInputs_triggerReference(t *testing.T) {
	state := newRunState()
	ctx := expression.EvalContext{Trigger: map[string]any{"name": "Alice"}}
	got, err := ResolveInputs(map[string]any{"greeting": "hello {{ $trigger.name }}"}, "n1", ctx, &state)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if got["greeting"] != "hello Alice" {
		t.Errorf("got %+v", got)
	}
}

func TestResolveInputs_nodeReference(t *testing.T) {
	state := newRunState()
	state.NodeOutputs["n0"] = map[string]any{"value": 42}
	ctx := expression.EvalContext{Nodes: state.NodeOutputs}
	got, err := ResolveInputs(map[string]any{"x": "{{ $nodes.n0.value }}"}, "n1", ctx, &state)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if got["x"] != 42 {
		t.Errorf("got %+v", got)
	}
}

func TestResolveInputs_nestedMap(t *testing.T) {
	state := newRunState()
	ctx := expression.EvalContext{Trigger: map[string]any{"id": 7}}
	got, err := ResolveInputs(map[string]any{
		"obj": map[string]any{
			"id":    "{{ $trigger.id }}",
			"label": "static",
		},
	}, "n1", ctx, &state)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	want := map[string]any{
		"obj": map[string]any{"id": 7, "label": "static"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestResolveInputs_arrayOfExpressions(t *testing.T) {
	state := newRunState()
	ctx := expression.EvalContext{Trigger: map[string]any{"a": 1, "b": 2}}
	got, err := ResolveInputs(map[string]any{
		"items": []any{"{{ $trigger.a }}", "{{ $trigger.b }}", "literal"},
	}, "n1", ctx, &state)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	items := got["items"].([]any)
	if items[0] != 1 || items[1] != 2 || items[2] != "literal" {
		t.Errorf("got %+v", items)
	}
}

func newRunState() RunState {
	return RunState{
		NodeStatus:  map[string]string{},
		NodeOutputs: map[string]map[string]any{},
		NodeInputs:  map[string]map[string]any{},
		NodePort:    map[string]string{},
	}
}
