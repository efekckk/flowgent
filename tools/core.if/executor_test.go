package coreif

import (
	"context"
	"testing"
)

func TestExecute_numericGreaterThan(t *testing.T) {
	e := New()
	res, err := e.Execute(context.Background(), map[string]any{
		"left": 150, "op": ">", "right": 100,
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if res.Port != "true" || res.Output["matched"] != true {
		t.Errorf("port=%s output=%+v", res.Port, res.Output)
	}
}

func TestExecute_numericLessThanFalse(t *testing.T) {
	e := New()
	res, err := e.Execute(context.Background(), map[string]any{
		"left": 50, "op": ">", "right": 100,
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if res.Port != "false" || res.Output["matched"] != false {
		t.Errorf("port=%s output=%+v", res.Port, res.Output)
	}
}

func TestExecute_stringEquality(t *testing.T) {
	e := New()
	res, err := e.Execute(context.Background(), map[string]any{
		"left": "hello", "op": "==", "right": "hello",
	})
	if err != nil || res.Port != "true" {
		t.Errorf("unexpected: %v port=%s", err, res.Port)
	}
}

func TestExecute_containsSubstring(t *testing.T) {
	e := New()
	res, err := e.Execute(context.Background(), map[string]any{
		"left": "alice@example.com", "op": "contains", "right": "@",
	})
	if err != nil || res.Port != "true" {
		t.Errorf("unexpected: %v port=%s", err, res.Port)
	}
}

func TestExecute_unsupportedOp(t *testing.T) {
	e := New()
	_, err := e.Execute(context.Background(), map[string]any{
		"left": 1, "op": "between", "right": 2,
	})
	if err == nil {
		t.Fatalf("expected error")
	}
}
