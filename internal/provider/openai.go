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

// OpenAI adapts the OpenAI Chat Completions API. The base URL is configurable
// so tests can point at httptest servers; production uses
// https://api.openai.com/v1.
type OpenAI struct {
	baseURL string
	apiKey  string
	client  *http.Client
}

func NewOpenAI(baseURL, apiKey string) *OpenAI {
	return &OpenAI{
		baseURL: baseURL,
		apiKey:  apiKey,
		client:  &http.Client{Timeout: 60 * time.Second},
	}
}

type openaiMessage struct {
	Role       string           `json:"role"`
	Content    any              `json:"content,omitempty"`
	ToolCalls  []openaiToolCall `json:"tool_calls,omitempty"`
	ToolCallID string           `json:"tool_call_id,omitempty"`
	Name       string           `json:"name,omitempty"`
}

type openaiToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

type openaiTool struct {
	Type     string         `json:"type"`
	Function openaiToolFunc `json:"function"`
}

type openaiToolFunc struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

type openaiRequest struct {
	Model       string          `json:"model"`
	Messages    []openaiMessage `json:"messages"`
	Tools       []openaiTool    `json:"tools,omitempty"`
	Temperature float64         `json:"temperature,omitempty"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
}

type openaiResponse struct {
	ID      string `json:"id"`
	Model   string `json:"model"`
	Choices []struct {
		FinishReason string        `json:"finish_reason"`
		Message      openaiMessage `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

func (o *OpenAI) Chat(ctx context.Context, req ChatRequest) (ChatResponse, error) {
	body := openaiRequest{
		Model:       req.Model,
		Messages:    toOpenAIMessages(req.Messages),
		Tools:       toOpenAITools(req.Tools),
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return ChatResponse{}, fmt.Errorf("openai: marshal: %w", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, "POST", o.baseURL+"/chat/completions", bytes.NewReader(raw))
	if err != nil {
		return ChatResponse{}, fmt.Errorf("openai: build request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+o.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := o.client.Do(httpReq)
	if err != nil {
		return ChatResponse{}, fmt.Errorf("openai: http: %w", err)
	}
	defer resp.Body.Close()
	respBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return ChatResponse{}, fmt.Errorf("openai: http %d: %s", resp.StatusCode, string(respBytes))
	}
	var parsed openaiResponse
	if err := json.Unmarshal(respBytes, &parsed); err != nil {
		return ChatResponse{}, fmt.Errorf("openai: parse: %w", err)
	}
	if len(parsed.Choices) == 0 {
		return ChatResponse{}, fmt.Errorf("openai: no choices in response")
	}
	choice := parsed.Choices[0]
	content := ""
	if s, ok := choice.Message.Content.(string); ok {
		content = s
	}
	stop := normaliseStop(choice.FinishReason)
	out := ChatResponse{
		Content:    content,
		StopReason: stop,
		Model:      parsed.Model,
		Usage: TokenUsage{
			PromptTokens:     parsed.Usage.PromptTokens,
			CompletionTokens: parsed.Usage.CompletionTokens,
			TotalTokens:      parsed.Usage.TotalTokens,
		},
	}
	for _, tc := range choice.Message.ToolCalls {
		out.ToolCalls = append(out.ToolCalls, ToolCall{
			ID:        tc.ID,
			Name:      tc.Function.Name,
			Arguments: json.RawMessage(tc.Function.Arguments),
		})
	}
	return out, nil
}

func toOpenAIMessages(in []Message) []openaiMessage {
	out := make([]openaiMessage, 0, len(in))
	for _, m := range in {
		om := openaiMessage{Role: m.Role}
		if m.Content != "" {
			om.Content = m.Content
		}
		if m.Role == "tool" {
			om.ToolCallID = m.ToolCallID
			om.Name = m.Name
			om.Content = string(m.ToolResults)
		}
		for _, tc := range m.ToolCalls {
			otc := openaiToolCall{ID: tc.ID, Type: "function"}
			otc.Function.Name = tc.Name
			otc.Function.Arguments = string(tc.Arguments)
			om.ToolCalls = append(om.ToolCalls, otc)
		}
		out = append(out, om)
	}
	return out
}

func toOpenAITools(in []ToolDefinition) []openaiTool {
	out := make([]openaiTool, 0, len(in))
	for _, td := range in {
		out = append(out, openaiTool{
			Type: "function",
			Function: openaiToolFunc{
				Name:        td.Name,
				Description: td.Description,
				Parameters:  td.InputSchema,
			},
		})
	}
	return out
}

func normaliseStop(s string) string {
	switch s {
	case "stop":
		return "stop"
	case "tool_calls":
		return "tool_use"
	case "length":
		return "max_tokens"
	}
	return s
}
