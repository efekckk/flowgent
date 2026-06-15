package coreloop

import (
	"context"
	"testing"
)

func TestExecute_returnsMetadata(t *testing.T) {
	e := New()
	res, err := e.Execute(context.Background(), map[string]any{
		"items": []any{1, 2, 3},
		"body":  []any{"b1"},
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if res.Port != "metadata" {
		t.Errorf("port: %s", res.Port)
	}
	if total, _ := res.Output["total"].(int); total != 3 {
		t.Errorf("total: %+v", res.Output)
	}
}

func TestExecute_emptyItemsStillReturns(t *testing.T) {
	e := New()
	res, err := e.Execute(context.Background(), map[string]any{
		"items": []any{},
		"body":  []any{"b1"},
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if total, _ := res.Output["total"].(int); total != 0 {
		t.Errorf("total: %+v", res.Output)
	}
}

func TestExecute_missingItemsIsError(t *testing.T) {
	e := New()
	_, err := e.Execute(context.Background(), map[string]any{"body": []any{"b1"}})
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestExecute_missingBodyIsError(t *testing.T) {
	e := New()
	_, err := e.Execute(context.Background(), map[string]any{"items": []any{1}})
	if err == nil {
		t.Fatalf("expected error")
	}
}
