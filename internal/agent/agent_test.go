package agent_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/efekckk/flowgent/internal/agent"
	"github.com/efekckk/flowgent/internal/provider"
)

func TestAgent_proposeWorkflow_validOnFirstTry(t *testing.T) {
	mock := provider.NewMock()
	mock.Reply(provider.ChatResponse{
		Content: "Here is a workflow.",
		ToolCalls: []provider.ToolCall{{
			ID:   "call_1",
			Name: "propose_workflow",
			Arguments: []byte(`{
				"name": "demo",
				"nodes": [{"id":"n1","tool":"core.set","params":{"values":{}}}],
				"edges": []
			}`),
		}},
		StopReason: "tool_use",
	})

	a := agent.New(agent.Deps{
		Provider:   mock,
		KnownTools: map[string]struct{}{"core.set": {}},
		MaxRetries: 3,
	})
	out, err := a.Run(context.Background(), agent.RunRequest{
		Model:           "gpt-4o",
		UserMessage:     "Build me a workflow",
		History:         nil,
		CurrentWorkflow: nil,
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if out.ProposedWorkflow == nil {
		t.Fatalf("proposed missing")
	}
	if !strings.Contains(string(out.ProposedWorkflow), "core.set") {
		t.Errorf("workflow: %s", string(out.ProposedWorkflow))
	}
	if out.ToolName != "propose_workflow" {
		t.Errorf("tool: %s", out.ToolName)
	}
}

func TestAgent_retriesOnValidationError(t *testing.T) {
	mock := provider.NewMock()
	// First reply: unknown tool
	mock.Reply(provider.ChatResponse{
		Content: "Try one",
		ToolCalls: []provider.ToolCall{{
			ID: "call_1", Name: "propose_workflow",
			Arguments: []byte(`{
				"name": "demo",
				"nodes": [{"id":"n1","tool":"bad.tool","params":{}}],
				"edges": []
			}`),
		}},
		StopReason: "tool_use",
	})
	// Second reply: valid
	mock.Reply(provider.ChatResponse{
		Content: "Fixed",
		ToolCalls: []provider.ToolCall{{
			ID: "call_2", Name: "propose_workflow",
			Arguments: []byte(`{
				"name": "demo",
				"nodes": [{"id":"n1","tool":"core.set","params":{"values":{}}}],
				"edges": []
			}`),
		}},
		StopReason: "tool_use",
	})

	a := agent.New(agent.Deps{
		Provider:   mock,
		KnownTools: map[string]struct{}{"core.set": {}},
		MaxRetries: 3,
	})
	out, err := a.Run(context.Background(), agent.RunRequest{
		Model:       "gpt-4o",
		UserMessage: "Build it",
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if out.ProposedWorkflow == nil {
		t.Fatalf("no proposal")
	}
	if !strings.Contains(string(out.ProposedWorkflow), "core.set") {
		t.Errorf("workflow: %s", string(out.ProposedWorkflow))
	}
	if len(mock.Calls()) != 2 {
		t.Errorf("calls: %d", len(mock.Calls()))
	}
}

func TestAgent_givesUpAfterMaxRetries(t *testing.T) {
	mock := provider.NewMock()
	for i := 0; i < 5; i++ {
		mock.Reply(provider.ChatResponse{
			Content: "Try",
			ToolCalls: []provider.ToolCall{{
				ID: "call_", Name: "propose_workflow",
				Arguments: []byte(`{"name":"","nodes":[],"edges":[]}`),
			}},
			StopReason: "tool_use",
		})
	}

	a := agent.New(agent.Deps{
		Provider:   mock,
		KnownTools: map[string]struct{}{"core.set": {}},
		MaxRetries: 3,
	})
	_, err := a.Run(context.Background(), agent.RunRequest{
		Model:       "gpt-4o",
		UserMessage: "Build it",
	})
	if err == nil {
		t.Fatalf("expected exhausted retries error")
	}
	if len(mock.Calls()) != 3 {
		t.Errorf("calls: %d (want 3)", len(mock.Calls()))
	}
}

func TestAgent_plainTextResponseReturnsContent(t *testing.T) {
	mock := provider.NewMock()
	mock.Reply(provider.ChatResponse{
		Content:    "I'd need more details about what you want.",
		StopReason: "stop",
	})

	a := agent.New(agent.Deps{
		Provider:   mock,
		KnownTools: map[string]struct{}{"core.set": {}},
		MaxRetries: 3,
	})
	out, err := a.Run(context.Background(), agent.RunRequest{
		Model:       "gpt-4o",
		UserMessage: "hi",
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if out.AssistantText != "I'd need more details about what you want." {
		t.Errorf("text: %s", out.AssistantText)
	}
	if out.ProposedWorkflow != nil {
		t.Errorf("unexpected proposal: %s", string(out.ProposedWorkflow))
	}
	_ = json.RawMessage{} // keep encoding/json import live
}
