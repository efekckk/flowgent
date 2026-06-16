// Package telegramsend implements "telegram.send_message" — Telegram Bot API
// sendMessage. The credential payload carries the bot_token; the workflow
// node supplies chat_id and message text. Transport errors deliberately do
// not include the request URL because the bot_token is embedded in the path
// and must never reach logs or the run viewer.
package telegramsend

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/efekckk/flowgent/internal/executor"
	"github.com/efekckk/flowgent/internal/registry"
)

const defaultBase = "https://api.telegram.org"

type Executor struct {
	baseURL string
	client  *http.Client
}

func New() *Executor { return newWithBase(defaultBase) }

func newWithBase(base string) *Executor {
	return &Executor{baseURL: base, client: &http.Client{Timeout: 15 * time.Second}}
}

func (e *Executor) Execute(ctx context.Context, input map[string]any) (registry.ExecuteResult, error) {
	chatID, err := stringifyChatID(input["chat_id"])
	if err != nil {
		return registry.ExecuteResult{}, fmt.Errorf("telegram.send_message: %w", err)
	}
	text, _ := input["text"].(string)
	if text == "" {
		return registry.ExecuteResult{}, fmt.Errorf("telegram.send_message: missing \"text\"")
	}
	cred, ok := input["__credential"].(map[string]any)
	if !ok {
		return registry.ExecuteResult{}, fmt.Errorf("telegram.send_message: missing credential")
	}
	botToken, _ := cred["bot_token"].(string)
	if botToken == "" {
		return registry.ExecuteResult{}, fmt.Errorf("telegram.send_message: credential missing \"bot_token\"")
	}

	payload := map[string]any{"chat_id": chatID, "text": text}
	if pm, _ := input["parse_mode"].(string); pm != "" {
		payload["parse_mode"] = pm
	}
	body, _ := json.Marshal(payload)

	url := e.baseURL + "/bot" + botToken + "/sendMessage"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		// Do not wrap err — *url.Error embeds the URL which contains the bot_token.
		return registry.ExecuteResult{}, fmt.Errorf("telegram.send_message: build request failed")
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := e.client.Do(req)
	if err != nil {
		// Same reason — transport errors include the URL with bot_token in the message.
		return registry.ExecuteResult{}, fmt.Errorf("telegram.send_message: http transport error")
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}()

	switch {
	case resp.StatusCode == http.StatusTooManyRequests:
		return registry.ExecuteResult{}, fmt.Errorf("telegram.send_message: http %d: %w", resp.StatusCode, executor.ErrRateLimited)
	case resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden:
		return registry.ExecuteResult{}, fmt.Errorf("telegram.send_message: http %d: %w", resp.StatusCode, executor.ErrAuthFailed)
	case resp.StatusCode >= 500:
		return registry.ExecuteResult{}, fmt.Errorf("telegram.send_message: http %d: %w", resp.StatusCode, executor.ErrTransient5xx)
	case resp.StatusCode >= 400:
		return registry.ExecuteResult{}, fmt.Errorf("telegram.send_message: http %d: %w", resp.StatusCode, executor.ErrValidation)
	}

	var parsed struct {
		OK     bool `json:"ok"`
		Result struct {
			MessageID int64 `json:"message_id"`
		} `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return registry.ExecuteResult{}, fmt.Errorf("telegram.send_message: parse response failed")
	}
	if !parsed.OK {
		return registry.ExecuteResult{}, fmt.Errorf("telegram.send_message: api rejected request: %w", executor.ErrValidation)
	}

	return registry.ExecuteResult{
		Output: map[string]any{
			"message_id": parsed.Result.MessageID,
			"ok":         parsed.OK,
		},
		Port: "main",
	}, nil
}

// stringifyChatID accepts the JSON-typical shapes for a chat id (string,
// float64, int, int64) and produces the canonical string form Telegram
// expects. Returns "" with an error when the value is absent or of an
// unsupported type.
func stringifyChatID(v any) (string, error) {
	switch x := v.(type) {
	case string:
		if x == "" {
			return "", fmt.Errorf("missing \"chat_id\"")
		}
		return x, nil
	case float64:
		// JSON unmarshals numbers as float64 by default; only accept integers.
		if x != float64(int64(x)) {
			return "", fmt.Errorf("\"chat_id\" must be an integer")
		}
		return strconv.FormatInt(int64(x), 10), nil
	case int:
		return strconv.Itoa(x), nil
	case int64:
		return strconv.FormatInt(x, 10), nil
	case nil:
		return "", fmt.Errorf("missing \"chat_id\"")
	default:
		return "", fmt.Errorf("\"chat_id\" must be a string or integer")
	}
}
