package corecode

import (
	"context"
	"testing"
	"time"
)

func TestExecute_returnsObject(t *testing.T) {
	e := New()
	res, err := e.Execute(context.Background(), map[string]any{
		"code": `return { hello: "world", n: 1 + 2 };`,
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if res.Output["hello"] != "world" {
		t.Errorf("hello: %+v", res.Output)
	}
	if res.Output["n"] != int64(3) && res.Output["n"] != float64(3) {
		t.Errorf("n: %+v", res.Output["n"])
	}
}

func TestExecute_readsInput(t *testing.T) {
	e := New()
	res, err := e.Execute(context.Background(), map[string]any{
		"code":  `return { doubled: input.value * 2 };`,
		"value": 21,
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	v := res.Output["doubled"]
	if v != int64(42) && v != float64(42) {
		t.Errorf("doubled: %+v", v)
	}
}

func TestExecute_syntaxErrorBubbles(t *testing.T) {
	e := New()
	_, err := e.Execute(context.Background(), map[string]any{
		"code": `return invalid_;`,
	})
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestExecute_missingCodeIsError(t *testing.T) {
	e := New()
	_, err := e.Execute(context.Background(), map[string]any{})
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestExecute_cpuTimeoutInterrupts(t *testing.T) {
	e := New()
	start := time.Now()
	_, err := e.Execute(context.Background(), map[string]any{
		"code": `while (true) { var x = 1; } return null;`,
	})
	if err == nil {
		t.Fatalf("expected timeout error")
	}
	if elapsed := time.Since(start); elapsed > 1500*time.Millisecond {
		t.Errorf("timeout took too long: %v", elapsed)
	}
}

func TestExecute_outputSizeLimitEnforced(t *testing.T) {
	e := New()
	_, err := e.Execute(context.Background(), map[string]any{
		"code": `var s = "x".repeat(2 * 1024 * 1024); return { big: s };`,
	})
	if err == nil {
		t.Fatalf("expected oversized-output error")
	}
}

func TestExecute_snippetSizeLimitEnforced(t *testing.T) {
	e := New()
	huge := make([]byte, 60*1024)
	for i := range huge {
		huge[i] = 'a'
	}
	_, err := e.Execute(context.Background(), map[string]any{
		"code": "/* " + string(huge) + " */ return null;",
	})
	if err == nil {
		t.Fatalf("expected snippet-size error")
	}
}
