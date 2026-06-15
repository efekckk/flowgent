package coremerge

import (
	"context"
	"reflect"
	"testing"
)

func TestExecute_appendCollectsAll(t *testing.T) {
	e := New()
	res, err := e.Execute(context.Background(), map[string]any{
		"mode": "append",
		"inputs": []any{
			map[string]any{"a": 1},
			map[string]any{"b": 2},
		},
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	items, _ := res.Output["items"].([]any)
	if len(items) != 2 {
		t.Errorf("items: %+v", items)
	}
}

func TestExecute_mergeObjectsShallowMerges(t *testing.T) {
	e := New()
	res, err := e.Execute(context.Background(), map[string]any{
		"mode": "merge_objects",
		"inputs": []any{
			map[string]any{"a": 1, "shared": "first"},
			map[string]any{"b": 2, "shared": "second"},
		},
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	want := map[string]any{"a": 1, "b": 2, "shared": "second"}
	if !reflect.DeepEqual(res.Output, want) {
		t.Errorf("got %+v", res.Output)
	}
}

func TestExecute_firstToCompleteReturnsFirstNonNil(t *testing.T) {
	e := New()
	res, err := e.Execute(context.Background(), map[string]any{
		"mode": "first_to_complete",
		"inputs": []any{
			nil,
			map[string]any{"hit": true},
			map[string]any{"also": true},
		},
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if res.Output["hit"] != true {
		t.Errorf("got %+v", res.Output)
	}
}

func TestExecute_unknownModeIsError(t *testing.T) {
	e := New()
	_, err := e.Execute(context.Background(), map[string]any{"mode": "bogus", "inputs": []any{}})
	if err == nil {
		t.Fatalf("expected error")
	}
}
