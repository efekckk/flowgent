package corewait

import (
	"context"
	"testing"
	"time"
)

func TestExecute_waits(t *testing.T) {
	e := New()
	start := time.Now()
	res, err := e.Execute(context.Background(), map[string]any{"ms": float64(50)})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if time.Since(start) < 45*time.Millisecond {
		t.Errorf("did not actually wait")
	}
	if res.Port != "main" {
		t.Errorf("port: %s", res.Port)
	}
	if v, _ := res.Output["waited_ms"]; v != int64(50) {
		t.Errorf("waited_ms: %v", v)
	}
}

func TestExecute_invalidInput(t *testing.T) {
	e := New()
	_, err := e.Execute(context.Background(), map[string]any{"ms": "not-a-number"})
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestExecute_cancelled(t *testing.T) {
	e := New()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := e.Execute(ctx, map[string]any{"ms": float64(1000)})
	if err == nil {
		t.Fatalf("expected error on cancelled ctx")
	}
}
