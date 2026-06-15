// Package provider defines the chat provider abstraction. Implementations
// adapt vendor-specific HTTP APIs (OpenAI, Anthropic, ...) to a shared
// shape: take a list of normalised messages + tool definitions, return
// content + tool calls + usage.
package provider

import (
	"context"
	"encoding/json"
)

type ChatProvider interface {
	Chat(ctx context.Context, req ChatRequest) (ChatResponse, error)
}

type ChatRequest struct {
	Model       string
	Messages    []Message
	Tools       []ToolDefinition
	Temperature float64
	MaxTokens   int
}

type Message struct {
	Role        string          // "system" | "user" | "assistant" | "tool"
	Content     string
	ToolCalls   []ToolCall      // assistant turns with tool calls
	ToolCallID  string          // tool-result messages reference an assistant tool_call's ID
	ToolResults json.RawMessage // tool-result payload
	Name        string          // optional tool name for tool messages
}

type ToolDefinition struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"input_schema"`
}

type ToolCall struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

type ChatResponse struct {
	Content    string
	ToolCalls  []ToolCall
	Usage      TokenUsage
	StopReason string // "stop" | "tool_use" | "max_tokens"
	Model      string
}

type TokenUsage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}
