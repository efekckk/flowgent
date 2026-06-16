package telegramsend

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/efekckk/flowgent/internal/executor"
)

func TestExecute_postsToBotAPI(t *testing.T) {
	type capture struct {
		path string
		body map[string]any
		ctyp string
	}
	got := make(chan capture, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		var parsed map[string]any
		_ = json.Unmarshal(b, &parsed)
		got <- capture{path: r.URL.Path, body: parsed, ctyp: r.Header.Get("Content-Type")}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"result":{"message_id":42}}`))
	}))
	defer srv.Close()

	e := newWithBase(srv.URL)
	res, err := e.Execute(context.Background(), map[string]any{
		"chat_id":      "12345",
		"text":         "Hello",
		"__credential": map[string]any{"bot_token": "tok-xyz", "__type": "telegram_bot"},
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	select {
	case c := <-got:
		if !strings.HasSuffix(c.path, "/bottok-xyz/sendMessage") {
			t.Errorf("path: %s", c.path)
		}
		if c.body["text"] != "Hello" || c.body["chat_id"] != "12345" {
			t.Errorf("body: %+v", c.body)
		}
		if c.ctyp != "application/json" {
			t.Errorf("content-type: %q", c.ctyp)
		}
	default:
		t.Fatalf("handler never received request")
	}
	mid, _ := res.Output["message_id"].(int64)
	if mid != 42 {
		t.Errorf("message_id: %+v (%T)", res.Output["message_id"], res.Output["message_id"])
	}
	if res.Output["ok"] != true {
		t.Errorf("ok: %+v", res.Output["ok"])
	}
}

func TestExecute_parseModeIsForwarded(t *testing.T) {
	got := make(chan map[string]any, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		var parsed map[string]any
		_ = json.Unmarshal(b, &parsed)
		got <- parsed
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"result":{"message_id":1}}`))
	}))
	defer srv.Close()

	e := newWithBase(srv.URL)
	_, err := e.Execute(context.Background(), map[string]any{
		"chat_id":      "1",
		"text":         "*bold*",
		"parse_mode":   "MarkdownV2",
		"__credential": map[string]any{"bot_token": "t", "__type": "telegram_bot"},
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	body := <-got
	if body["parse_mode"] != "MarkdownV2" {
		t.Errorf("parse_mode: %+v", body["parse_mode"])
	}
}

func TestExecute_missingChatIsError(t *testing.T) {
	e := New()
	_, err := e.Execute(context.Background(), map[string]any{
		"text":         "hi",
		"__credential": map[string]any{"bot_token": "x", "__type": "telegram_bot"},
	})
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestExecute_missingTextIsError(t *testing.T) {
	e := New()
	_, err := e.Execute(context.Background(), map[string]any{
		"chat_id":      "1",
		"__credential": map[string]any{"bot_token": "x", "__type": "telegram_bot"},
	})
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestExecute_missingCredentialIsError(t *testing.T) {
	e := New()
	_, err := e.Execute(context.Background(), map[string]any{"chat_id": "1", "text": "x"})
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestExecute_429IsRateLimited(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	e := newWithBase(srv.URL)
	_, err := e.Execute(context.Background(), map[string]any{
		"chat_id": "1", "text": "x",
		"__credential": map[string]any{"bot_token": "t", "__type": "telegram_bot"},
	})
	if err == nil || !errors.Is(err, executor.ErrRateLimited) {
		t.Fatalf("expected ErrRateLimited, got %v", err)
	}
}

func TestExecute_401IsAuthFailed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	e := newWithBase(srv.URL)
	_, err := e.Execute(context.Background(), map[string]any{
		"chat_id": "1", "text": "x",
		"__credential": map[string]any{"bot_token": "t", "__type": "telegram_bot"},
	})
	if err == nil || !errors.Is(err, executor.ErrAuthFailed) {
		t.Fatalf("expected ErrAuthFailed, got %v", err)
	}
}

func TestExecute_5xxIsTransient(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer srv.Close()

	e := newWithBase(srv.URL)
	_, err := e.Execute(context.Background(), map[string]any{
		"chat_id": "1", "text": "x",
		"__credential": map[string]any{"bot_token": "t", "__type": "telegram_bot"},
	})
	if err == nil || !errors.Is(err, executor.ErrTransient5xx) {
		t.Fatalf("expected ErrTransient5xx, got %v", err)
	}
}
