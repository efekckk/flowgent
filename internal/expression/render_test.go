package expression

import "testing"

func TestRender_literal(t *testing.T) {
	got, err := RenderValue("plain text", EvalContext{})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if got != "plain text" {
		t.Errorf("want literal, got %v", got)
	}
}

func TestRender_singleExpressionPreservesType(t *testing.T) {
	ctx := EvalContext{Trigger: map[string]any{"port": 8080}}
	got, err := RenderValue("{{ $trigger.port }}", ctx)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if got != 8080 {
		t.Errorf("want int 8080, got %T %v", got, got)
	}
}

func TestRender_embeddedExpressionConcatenated(t *testing.T) {
	ctx := EvalContext{Trigger: map[string]any{"name": "Alice"}}
	got, err := RenderValue("hello, {{ $trigger.name }}!", ctx)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if got != "hello, Alice!" {
		t.Errorf("want \"hello, Alice!\", got %v", got)
	}
}

func TestRender_multipleExpressions(t *testing.T) {
	ctx := EvalContext{
		Trigger: map[string]any{"a": 3, "b": 4},
	}
	got, err := RenderValue("sum={{ $trigger.a + $trigger.b }} avg={{ ($trigger.a + $trigger.b) / 2 }}", ctx)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	// expr-lang treats `/` as float division, so (3+4)/2 == 3.5
	if got != "sum=7 avg=3.5" {
		t.Errorf("got %v", got)
	}
}

func TestRender_passThroughNonString(t *testing.T) {
	got, err := RenderValue(42, EvalContext{})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if got != 42 {
		t.Errorf("want 42, got %v", got)
	}
}

func TestRender_evaluationError(t *testing.T) {
	_, err := RenderValue("{{ $trigger.missing }}", EvalContext{Trigger: map[string]any{}})
	if err == nil {
		t.Fatalf("expected error")
	}
}
