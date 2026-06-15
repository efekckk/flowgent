// Package registry loads tool manifests from disk and exposes them to the
// executor and (later) the AI agent. A manifest pairs a JSON descriptor
// (this package) with a Go executor (registered separately at link time).
package registry

import (
	"encoding/json"
	"fmt"
	"regexp"
)

type Manifest struct {
	Slug        string          `json:"slug"`
	Version     string          `json:"version"`
	DisplayName string          `json:"display_name"`
	Category    string          `json:"category"`
	Description string          `json:"description"`
	Credentials *CredentialSpec `json:"credentials,omitempty"`
	Inputs      json.RawMessage `json:"inputs,omitempty"`
	Outputs     json.RawMessage `json:"outputs,omitempty"`
	Retry       RetryPolicy     `json:"retry,omitempty"`
	RateLimit   *RateLimit      `json:"rate_limit,omitempty"`
}

type CredentialSpec struct {
	Type     string `json:"type"`
	Required bool   `json:"required"`
}

type RetryPolicy struct {
	MaxAttempts     int      `json:"max_attempts"`
	Backoff         string   `json:"backoff"`
	BaseMs          int      `json:"base_ms"`
	RetryableErrors []string `json:"retryable_errors"`
}

type RateLimit struct {
	PerSecond int `json:"per_second"`
	Burst     int `json:"burst"`
}

// slugPattern matches "namespace.action" — lowercase letters, digits, dots,
// underscores. The leading char is a letter; "." separates segments.
var slugPattern = regexp.MustCompile(`^[a-z][a-z0-9_]*(\.[a-z][a-z0-9_]*)+$`)

func ParseManifest(b []byte) (Manifest, error) {
	var m Manifest
	if err := json.Unmarshal(b, &m); err != nil {
		return Manifest{}, fmt.Errorf("registry: parse manifest: %w", err)
	}
	if m.Slug == "" {
		return Manifest{}, fmt.Errorf("registry: manifest missing slug")
	}
	if !slugPattern.MatchString(m.Slug) {
		return Manifest{}, fmt.Errorf("registry: invalid slug format %q (want namespace.action)", m.Slug)
	}
	if m.Version == "" {
		return Manifest{}, fmt.Errorf("registry: manifest %q missing version", m.Slug)
	}
	if m.DisplayName == "" {
		return Manifest{}, fmt.Errorf("registry: manifest %q missing display_name", m.Slug)
	}
	if m.Category == "" {
		return Manifest{}, fmt.Errorf("registry: manifest %q missing category", m.Slug)
	}
	if m.Description == "" {
		return Manifest{}, fmt.Errorf("registry: manifest %q missing description", m.Slug)
	}
	if m.Retry.MaxAttempts < 1 {
		m.Retry.MaxAttempts = 1
	}
	return m, nil
}
