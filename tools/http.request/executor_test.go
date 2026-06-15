package httprequest

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

func TestExecute_GET_returnsParsedJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"hello": "world"}`))
	}))
	defer srv.Close()

	e := New()
	res, err := e.Execute(context.Background(), map[string]any{
		"method": "GET",
		"url":    srv.URL,
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	body, _ := res.Output["body"].(map[string]any)
	if body["hello"] != "world" {
		t.Errorf("body: %+v", res.Output["body"])
	}
	if res.Output["status"] != 200 {
		t.Errorf("status: %v", res.Output["status"])
	}
}

func TestExecute_POSTWithJSONBody(t *testing.T) {
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &got)
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"ok": true}`))
	}))
	defer srv.Close()

	e := New()
	_, err := e.Execute(context.Background(), map[string]any{
		"method": "POST",
		"url":    srv.URL,
		"body":   map[string]any{"name": "Alice"},
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if got["name"] != "Alice" {
		t.Errorf("server got: %+v", got)
	}
}

func TestExecute_5xx_isTransient(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer srv.Close()

	e := New()
	_, err := e.Execute(context.Background(), map[string]any{"method": "GET", "url": srv.URL})
	if err == nil || !errors.Is(err, executor.ErrTransient5xx) {
		t.Fatalf("expected ErrTransient5xx, got %v", err)
	}
}

func TestExecute_429_isRateLimited(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	e := New()
	_, err := e.Execute(context.Background(), map[string]any{"method": "GET", "url": srv.URL})
	if err == nil || !errors.Is(err, executor.ErrRateLimited) {
		t.Fatalf("expected ErrRateLimited, got %v", err)
	}
}

func TestExecute_401_isAuth(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	e := New()
	_, err := e.Execute(context.Background(), map[string]any{"method": "GET", "url": srv.URL})
	if err == nil || !errors.Is(err, executor.ErrAuthFailed) {
		t.Fatalf("expected ErrAuthFailed, got %v", err)
	}
}

func TestExecute_invalidURL(t *testing.T) {
	e := New()
	_, err := e.Execute(context.Background(), map[string]any{"method": "GET", "url": "::not-a-url"})
	if err == nil {
		t.Fatalf("expected error")
	}
}
