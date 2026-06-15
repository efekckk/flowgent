package provider_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/efekckk/flowgent/internal/provider"
)

func TestAnthropic_textOnlyResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/messages") {
			t.Errorf("path: %s", r.URL.Path)
		}
		if r.Header.Get("anthropic-version") == "" {
			t.Errorf("missing anthropic-version header")
		}
		_ = r.Body.Close()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id":"x","type":"message","model":"claude-3-7-sonnet","role":"assistant",
			"content":[{"type":"text","text":"hello"}],
			"stop_reason":"end_turn",
			"usage":{"input_tokens":10,"output_tokens":3}
		}`))
	}))
	defer srv.Close()

	c := provider.NewAnthropic(srv.URL, "sk-ant-test")
	resp, err := c.Chat(context.Background(), provider.ChatRequest{
		Model:    "claude-3-7-sonnet",
		Messages: []provider.Message{{Role: "user", Content: "hi"}},
	})
	if err != nil {
		t.Fatalf("chat: %v", err)
	}
	if resp.Content != "hello" {
		t.Errorf("content: %q", resp.Content)
	}
	if resp.StopReason != "stop" {
		t.Errorf("stop: %s", resp.StopReason)
	}
	if resp.Usage.TotalTokens != 13 {
		t.Errorf("usage: %+v", resp.Usage)
	}
}

func TestAnthropic_toolCallResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id":"x","type":"message","model":"claude-3-7-sonnet","role":"assistant",
			"content":[
				{"type":"text","text":"Let me propose a workflow."},
				{"type":"tool_use","id":"tu_1","name":"propose_workflow","input":{"nodes":[]}}
			],
			"stop_reason":"tool_use",
			"usage":{"input_tokens":20,"output_tokens":40}
		}`))
	}))
	defer srv.Close()

	c := provider.NewAnthropic(srv.URL, "sk-ant-test")
	resp, err := c.Chat(context.Background(), provider.ChatRequest{
		Model:    "claude-3-7-sonnet",
		Messages: []provider.Message{{Role: "user", Content: "make a workflow"}},
		Tools: []provider.ToolDefinition{{
			Name: "propose_workflow", Description: "Submit a workflow", InputSchema: []byte(`{"type":"object"}`),
		}},
	})
	if err != nil {
		t.Fatalf("chat: %v", err)
	}
	if resp.Content != "Let me propose a workflow." {
		t.Errorf("content: %q", resp.Content)
	}
	if len(resp.ToolCalls) != 1 || resp.ToolCalls[0].Name != "propose_workflow" {
		t.Errorf("tool_calls: %+v", resp.ToolCalls)
	}
	if resp.StopReason != "tool_use" {
		t.Errorf("stop: %s", resp.StopReason)
	}
}
