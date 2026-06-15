// Package httprequest implements "http.request" — a generic HTTP client tool.
// Status codes are classified into executor sentinel errors so the engine's
// retry policy works without per-tool knowledge.
package httprequest

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/efekckk/flowgent/internal/executor"
	"github.com/efekckk/flowgent/internal/registry"
)

type Executor struct {
	client *http.Client
}

func New() *Executor {
	return &Executor{client: &http.Client{Timeout: 30 * time.Second}}
}

func (e *Executor) Execute(ctx context.Context, input map[string]any) (registry.ExecuteResult, error) {
	method, ok := input["method"].(string)
	if !ok || method == "" {
		return registry.ExecuteResult{}, fmt.Errorf("http.request: missing \"method\"")
	}
	method = strings.ToUpper(method)
	url, ok := input["url"].(string)
	if !ok || url == "" {
		return registry.ExecuteResult{}, fmt.Errorf("http.request: missing \"url\"")
	}

	var bodyReader io.Reader
	if rawBody, hasBody := input["body"]; hasBody && rawBody != nil {
		if method == "GET" || method == "DELETE" {
			return registry.ExecuteResult{}, fmt.Errorf("http.request: body not allowed for %s", method)
		}
		encoded, err := json.Marshal(rawBody)
		if err != nil {
			return registry.ExecuteResult{}, fmt.Errorf("http.request: encode body: %w", err)
		}
		bodyReader = bytes.NewReader(encoded)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return registry.ExecuteResult{}, fmt.Errorf("http.request: build: %w", err)
	}
	if bodyReader != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if hdrs, ok := input["headers"].(map[string]any); ok {
		for k, v := range hdrs {
			req.Header.Set(k, fmt.Sprint(v))
		}
	}

	resp, err := e.client.Do(req)
	if err != nil {
		return registry.ExecuteResult{}, fmt.Errorf("http.request: do: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return registry.ExecuteResult{}, fmt.Errorf("http.request: read body: %w", err)
	}

	if cls := classifyStatus(resp.StatusCode); cls != nil {
		return registry.ExecuteResult{}, fmt.Errorf("http.request: http %d: %w", resp.StatusCode, cls)
	}

	out := map[string]any{
		"status":  resp.StatusCode,
		"headers": flattenHeaders(resp.Header),
		"body":    parseBody(resp.Header.Get("Content-Type"), raw),
	}
	return registry.ExecuteResult{Output: out, Port: "main"}, nil
}

func classifyStatus(code int) error {
	switch {
	case code == http.StatusTooManyRequests:
		return executor.ErrRateLimited
	case code == http.StatusUnauthorized || code == http.StatusForbidden:
		return executor.ErrAuthFailed
	case code >= 500:
		return executor.ErrTransient5xx
	case code >= 400:
		return executor.ErrValidation
	default:
		return nil
	}
}

func parseBody(contentType string, raw []byte) any {
	if strings.Contains(strings.ToLower(contentType), "application/json") && len(raw) > 0 {
		var v any
		if err := json.Unmarshal(raw, &v); err == nil {
			return v
		}
	}
	return string(raw)
}

func flattenHeaders(h http.Header) map[string]any {
	out := make(map[string]any, len(h))
	for k, vs := range h {
		if len(vs) == 1 {
			out[k] = vs[0]
		} else {
			out[k] = vs
		}
	}
	return out
}
