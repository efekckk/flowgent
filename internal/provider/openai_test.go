package provider_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/efekckk/flowgent/internal/provider"
)

func TestOpenAI_textOnlyResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/chat/completions") {
			t.Errorf("path: %s", r.URL.Path)
		}
		_ = r.Body.Close()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id":"x","model":"gpt-4o","object":"chat.completion",
			"choices":[{"index":0,"finish_reason":"stop","message":{"role":"assistant","content":"hello back"}}],
			"usage":{"prompt_tokens":10,"completion_tokens":3,"total_tokens":13}
		}`))
	}))
	defer srv.Close()

	c := provider.NewOpenAI(srv.URL, "sk-test")
	resp, err := c.Chat(context.Background(), provider.ChatRequest{
		Model:    "gpt-4o",
		Messages: []provider.Message{{Role: "user", Content: "hi"}},
	})
	if err != nil {
		t.Fatalf("chat: %v", err)
	}
	if resp.Content != "hello back" {
		t.Errorf("content: %q", resp.Content)
	}
	if resp.StopReason != "stop" {
		t.Errorf("stop: %s", resp.StopReason)
	}
	if resp.Usage.TotalTokens != 13 {
		t.Errorf("usage: %+v", resp.Usage)
	}
}

func TestOpenAI_toolCallResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = r.Body.Close()
		var raw map[string]any
		_ = json.Unmarshal(body, &raw)
		tools, _ := raw["tools"].([]any)
		if len(tools) == 0 {
			t.Errorf("tools missing from request")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id":"x","model":"gpt-4o","object":"chat.completion",
			"choices":[{"index":0,"finish_reason":"tool_calls","message":{
				"role":"assistant","content":null,
				"tool_calls":[{
					"id":"call_1","type":"function",
					"function":{"name":"propose_workflow","arguments":"{\"nodes\":[]}"}
				}]
			}}],
			"usage":{"prompt_tokens":10,"completion_tokens":3,"total_tokens":13}
		}`))
	}))
	defer srv.Close()

	c := provider.NewOpenAI(srv.URL, "sk-test")
	resp, err := c.Chat(context.Background(), provider.ChatRequest{
		Model:    "gpt-4o",
		Messages: []provider.Message{{Role: "user", Content: "make a workflow"}},
		Tools: []provider.ToolDefinition{{
			Name: "propose_workflow", Description: "Submit a workflow", InputSchema: []byte(`{"type":"object"}`),
		}},
	})
	if err != nil {
		t.Fatalf("chat: %v", err)
	}
	if len(resp.ToolCalls) != 1 {
		t.Fatalf("tool_calls: %+v", resp.ToolCalls)
	}
	if resp.ToolCalls[0].Name != "propose_workflow" {
		t.Errorf("name: %s", resp.ToolCalls[0].Name)
	}
	if resp.StopReason != "tool_use" {
		t.Errorf("stop: %s", resp.StopReason)
	}
}

func TestOpenAI_apiError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"message":"bad key"}}`))
	}))
	defer srv.Close()

	c := provider.NewOpenAI(srv.URL, "sk-bad")
	_, err := c.Chat(context.Background(), provider.ChatRequest{
		Model: "gpt-4o", Messages: []provider.Message{{Role: "user", Content: "hi"}},
	})
	if err == nil {
		t.Fatalf("expected error")
	}
}
