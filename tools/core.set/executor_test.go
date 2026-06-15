package coreset

import (
	"context"
	"reflect"
	"testing"
)

func TestExecute_passesValuesThrough(t *testing.T) {
	e := New()
	res, err := e.Execute(context.Background(), map[string]any{
		"values": map[string]any{
			"name":  "Alice",
			"score": 42,
		},
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if res.Port != "main" {
		t.Errorf("port: %s", res.Port)
	}
	want := map[string]any{"name": "Alice", "score": 42}
	if !reflect.DeepEqual(res.Output, want) {
		t.Errorf("output: %+v", res.Output)
	}
}

func TestExecute_missingValuesIsError(t *testing.T) {
	e := New()
	_, err := e.Execute(context.Background(), map[string]any{})
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestExecute_emptyValuesIsAllowed(t *testing.T) {
	e := New()
	res, err := e.Execute(context.Background(), map[string]any{"values": map[string]any{}})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if len(res.Output) != 0 {
		t.Errorf("output: %+v", res.Output)
	}
}
