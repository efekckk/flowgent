package expression

import (
	"testing"
	"time"
)

func TestEvaluate_triggerField(t *testing.T) {
	ctx := EvalContext{Trigger: map[string]any{"amount": 150}}
	got, err := Evaluate("$trigger.amount", ctx)
	if err != nil {
		t.Fatalf("eval: %v", err)
	}
	if got != 150 {
		t.Errorf("want 150, got %v", got)
	}
}

func TestEvaluate_nodeOutputField(t *testing.T) {
	ctx := EvalContext{Nodes: map[string]map[string]any{
		"n1": {"body": "ok"},
	}}
	got, err := Evaluate("$nodes.n1.body", ctx)
	if err != nil {
		t.Fatalf("eval: %v", err)
	}
	if got != "ok" {
		t.Errorf("want \"ok\", got %v", got)
	}
}

func TestEvaluate_arithmetic(t *testing.T) {
	ctx := EvalContext{Trigger: map[string]any{"a": 3, "b": 4}}
	got, err := Evaluate("$trigger.a + $trigger.b", ctx)
	if err != nil {
		t.Fatalf("eval: %v", err)
	}
	if got != 7 {
		t.Errorf("want 7, got %v", got)
	}
}

func TestEvaluate_now(t *testing.T) {
	fixed := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	ctx := EvalContext{Now: fixed}
	got, err := Evaluate("$now.Year()", ctx)
	if err != nil {
		t.Fatalf("eval: %v", err)
	}
	if got != 2026 {
		t.Errorf("want 2026, got %v", got)
	}
}

func TestEvaluate_missingField(t *testing.T) {
	ctx := EvalContext{Trigger: map[string]any{}}
	_, err := Evaluate("$trigger.missing", ctx)
	if err == nil {
		t.Fatalf("expected error on missing field")
	}
}

func TestEvaluate_syntaxError(t *testing.T) {
	ctx := EvalContext{}
	_, err := Evaluate("$trigger.(invalid", ctx)
	if err == nil {
		t.Fatalf("expected error on syntax")
	}
}

func TestEvaluate_stringLiteralWithDotAccess(t *testing.T) {
	ctx := EvalContext{Trigger: map[string]any{"kind": "purchase"}}
	// The literal string "trigger.type" must not trigger missing-key validation;
	// only the actual $trigger.kind reference is a real field access.
	got, err := Evaluate(`"trigger.type" == $trigger.kind`, ctx)
	if err != nil {
		t.Fatalf("eval: %v", err)
	}
	if got != false {
		t.Errorf("want false (\"trigger.type\" != \"purchase\"), got %v", got)
	}
}
