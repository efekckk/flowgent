package slacksend

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/efekckk/flowgent/internal/executor"
)

func TestExecute_postsMessageToWebhook(t *testing.T) {
	var receivedBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &receivedBody)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()

	e := New()
	res, err := e.Execute(context.Background(), map[string]any{
		"text":         "Hello from Flowgent",
		"__credential": map[string]any{"url": srv.URL, "__type": "slack_webhook"},
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if receivedBody["text"] != "Hello from Flowgent" {
		t.Errorf("body: %+v", receivedBody)
	}
	if res.Output["ok"] != true || res.Output["status"] != 200 {
		t.Errorf("output: %+v", res.Output)
	}
}

func TestExecute_missingTextIsError(t *testing.T) {
	e := New()
	_, err := e.Execute(context.Background(), map[string]any{
		"__credential": map[string]any{"url": "http://example", "__type": "slack_webhook"},
	})
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestExecute_missingCredentialIsError(t *testing.T) {
	e := New()
	_, err := e.Execute(context.Background(), map[string]any{"text": "hi"})
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestExecute_429IsRateLimited(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	e := New()
	_, err := e.Execute(context.Background(), map[string]any{
		"text":         "hi",
		"__credential": map[string]any{"url": srv.URL, "__type": "slack_webhook"},
	})
	if err == nil || !errors.Is(err, executor.ErrRateLimited) {
		t.Fatalf("expected ErrRateLimited, got %v", err)
	}
}

func TestExecute_5xxIsTransient(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer srv.Close()

	e := New()
	_, err := e.Execute(context.Background(), map[string]any{
		"text":         "hi",
		"__credential": map[string]any{"url": srv.URL, "__type": "slack_webhook"},
	})
	if err == nil || !errors.Is(err, executor.ErrTransient5xx) {
		t.Fatalf("expected ErrTransient5xx, got %v", err)
	}
}
