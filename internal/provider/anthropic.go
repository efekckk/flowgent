package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Anthropic adapts the Anthropic Messages API. Note that this API uses a
// different shape than OpenAI: content blocks can be text or tool_use, and
// the request uses input_schema not parameters.
type Anthropic struct {
	baseURL string
	apiKey  string
	version string
	client  *http.Client
}

func NewAnthropic(baseURL, apiKey string) *Anthropic {
	return &Anthropic{
		baseURL: baseURL,
		apiKey:  apiKey,
		version: "2023-06-01",
		client:  &http.Client{Timeout: 60 * time.Second},
	}
}

type anthropicTool struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"input_schema"`
}

type anthropicMessage struct {
	Role    string                  `json:"role"`
	Content []anthropicContentBlock `json:"content"`
}

type anthropicContentBlock struct {
	Type      string          `json:"type"`
	Text      string          `json:"text,omitempty"`
	ID        string          `json:"id,omitempty"`
	Name      string          `json:"name,omitempty"`
	Input     json.RawMessage `json:"input,omitempty"`
	ToolUseID string          `json:"tool_use_id,omitempty"`
	Content   string          `json:"content,omitempty"` // tool_result
}

type anthropicRequest struct {
	Model     string             `json:"model"`
	Messages  []anthropicMessage `json:"messages"`
	Tools     []anthropicTool    `json:"tools,omitempty"`
	MaxTokens int                `json:"max_tokens"`
	System    string             `json:"system,omitempty"`
}

type anthropicResponse struct {
	ID         string                  `json:"id"`
	Model      string                  `json:"model"`
	Content    []anthropicContentBlock `json:"content"`
	StopReason string                  `json:"stop_reason"`
	Usage      struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

func (a *Anthropic) Chat(ctx context.Context, req ChatRequest) (ChatResponse, error) {
	system, messages := splitSystem(req.Messages)
	body := anthropicRequest{
		Model:     req.Model,
		Messages:  toAnthropicMessages(messages),
		Tools:     toAnthropicTools(req.Tools),
		MaxTokens: orDefault(req.MaxTokens, 4096),
		System:    system,
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return ChatResponse{}, fmt.Errorf("anthropic: marshal: %w", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, "POST", a.baseURL+"/messages", bytes.NewReader(raw))
	if err != nil {
		return ChatResponse{}, fmt.Errorf("anthropic: build request: %w", err)
	}
	httpReq.Header.Set("x-api-key", a.apiKey)
	httpReq.Header.Set("anthropic-version", a.version)
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := a.client.Do(httpReq)
	if err != nil {
		return ChatResponse{}, fmt.Errorf("anthropic: http: %w", err)
	}
	defer resp.Body.Close()
	respBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return ChatResponse{}, fmt.Errorf("anthropic: http %d: %s", resp.StatusCode, string(respBytes))
	}
	var parsed anthropicResponse
	if err := json.Unmarshal(respBytes, &parsed); err != nil {
		return ChatResponse{}, fmt.Errorf("anthropic: parse: %w", err)
	}
	out := ChatResponse{
		StopReason: normaliseAnthropicStop(parsed.StopReason),
		Model:      parsed.Model,
		Usage: TokenUsage{
			PromptTokens:     parsed.Usage.InputTokens,
			CompletionTokens: parsed.Usage.OutputTokens,
			TotalTokens:      parsed.Usage.InputTokens + parsed.Usage.OutputTokens,
		},
	}
	for _, block := range parsed.Content {
		switch block.Type {
		case "text":
			out.Content += block.Text
		case "tool_use":
			out.ToolCalls = append(out.ToolCalls, ToolCall{
				ID:        block.ID,
				Name:      block.Name,
				Arguments: block.Input,
			})
		}
	}
	return out, nil
}

func splitSystem(in []Message) (string, []Message) {
	system := ""
	rest := make([]Message, 0, len(in))
	for _, m := range in {
		if m.Role == "system" {
			if system != "" {
				system += "\n\n"
			}
			system += m.Content
			continue
		}
		rest = append(rest, m)
	}
	return system, rest
}

func toAnthropicMessages(in []Message) []anthropicMessage {
	out := make([]anthropicMessage, 0, len(in))
	for _, m := range in {
		role := m.Role
		if role == "tool" {
			role = "user"
		}
		blocks := []anthropicContentBlock{}
		if m.Content != "" {
			blocks = append(blocks, anthropicContentBlock{Type: "text", Text: m.Content})
		}
		for _, tc := range m.ToolCalls {
			blocks = append(blocks, anthropicContentBlock{
				Type:  "tool_use",
				ID:    tc.ID,
				Name:  tc.Name,
				Input: tc.Arguments,
			})
		}
		if m.Role == "tool" {
			blocks = append(blocks, anthropicContentBlock{
				Type:      "tool_result",
				ToolUseID: m.ToolCallID,
				Content:   string(m.ToolResults),
			})
		}
		out = append(out, anthropicMessage{Role: role, Content: blocks})
	}
	return out
}

func toAnthropicTools(in []ToolDefinition) []anthropicTool {
	out := make([]anthropicTool, 0, len(in))
	for _, td := range in {
		out = append(out, anthropicTool{
			Name:        td.Name,
			Description: td.Description,
			InputSchema: td.InputSchema,
		})
	}
	return out
}

func normaliseAnthropicStop(s string) string {
	switch s {
	case "end_turn":
		return "stop"
	case "tool_use":
		return "tool_use"
	case "max_tokens":
		return "max_tokens"
	}
	return s
}

func orDefault(v, def int) int {
	if v == 0 {
		return def
	}
	return v
}
