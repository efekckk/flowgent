// Package slacksend implements "slack.send_message" — posts a message to a
// Slack channel via an incoming webhook. The credential payload carries the
// webhook URL; the workflow node only supplies the message text.
package slacksend

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/efekckk/flowgent/internal/executor"
	"github.com/efekckk/flowgent/internal/registry"
)

type Executor struct {
	client *http.Client
}

func New() *Executor {
	return &Executor{client: &http.Client{Timeout: 15 * time.Second}}
}

func (e *Executor) Execute(ctx context.Context, input map[string]any) (registry.ExecuteResult, error) {
	text, _ := input["text"].(string)
	if text == "" {
		return registry.ExecuteResult{}, fmt.Errorf("slack.send_message: missing \"text\"")
	}
	cred, ok := input["__credential"].(map[string]any)
	if !ok {
		return registry.ExecuteResult{}, fmt.Errorf("slack.send_message: missing credential")
	}
	url, _ := cred["url"].(string)
	if url == "" {
		return registry.ExecuteResult{}, fmt.Errorf("slack.send_message: credential missing \"url\"")
	}

	body, _ := json.Marshal(map[string]any{"text": text})
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return registry.ExecuteResult{}, fmt.Errorf("slack.send_message: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := e.client.Do(req)
	if err != nil {
		return registry.ExecuteResult{}, fmt.Errorf("slack.send_message: http: %w", err)
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode == http.StatusTooManyRequests:
		return registry.ExecuteResult{}, fmt.Errorf("slack.send_message: http %d: %w", resp.StatusCode, executor.ErrRateLimited)
	case resp.StatusCode >= 500:
		return registry.ExecuteResult{}, fmt.Errorf("slack.send_message: http %d: %w", resp.StatusCode, executor.ErrTransient5xx)
	case resp.StatusCode >= 400:
		return registry.ExecuteResult{}, fmt.Errorf("slack.send_message: http %d: %w", resp.StatusCode, executor.ErrValidation)
	}

	return registry.ExecuteResult{
		Output: map[string]any{"ok": true, "status": resp.StatusCode},
		Port:   "main",
	}, nil
}
