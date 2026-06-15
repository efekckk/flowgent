package provider_test

import (
	"context"
	"testing"

	"github.com/efekckk/flowgent/internal/provider"
)

func TestMockProvider_returnsScriptedResponse(t *testing.T) {
	m := provider.NewMock()
	m.Reply(provider.ChatResponse{
		Content: "Hello!",
		ToolCalls: []provider.ToolCall{
			{Name: "propose_workflow", Arguments: []byte(`{"nodes":[]}`)},
		},
		StopReason: "tool_use",
	})
	resp, err := m.Chat(context.Background(), provider.ChatRequest{
		Messages: []provider.Message{{Role: "user", Content: "hi"}},
	})
	if err != nil {
		t.Fatalf("chat: %v", err)
	}
	if resp.Content != "Hello!" {
		t.Errorf("content: %s", resp.Content)
	}
	if len(resp.ToolCalls) != 1 || resp.ToolCalls[0].Name != "propose_workflow" {
		t.Errorf("tool calls: %+v", resp.ToolCalls)
	}
}

func TestMockProvider_recordsRequests(t *testing.T) {
	m := provider.NewMock()
	m.Reply(provider.ChatResponse{Content: "ok"})
	_, _ = m.Chat(context.Background(), provider.ChatRequest{
		Model:    "gpt-4o",
		Messages: []provider.Message{{Role: "user", Content: "hi"}},
	})
	if len(m.Calls()) != 1 {
		t.Errorf("calls: %d", len(m.Calls()))
	}
	if m.Calls()[0].Model != "gpt-4o" {
		t.Errorf("model: %s", m.Calls()[0].Model)
	}
}
