package registry

import (
	"strings"
	"testing"
)

func TestParseManifest_minimal(t *testing.T) {
	src := []byte(`{
		"slug": "core.wait",
		"version": "1.0.0",
		"display_name": "Wait",
		"category": "control",
		"description": "Pauses execution for N milliseconds.",
		"inputs": {"type": "object", "required": ["ms"], "properties": {"ms": {"type": "integer"}}},
		"outputs": {"type": "object"}
	}`)
	m, err := ParseManifest(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if m.Slug != "core.wait" || m.Version != "1.0.0" {
		t.Errorf("got %+v", m)
	}
	if m.Retry.MaxAttempts != 1 {
		t.Errorf("default MaxAttempts should be 1, got %d", m.Retry.MaxAttempts)
	}
}

func TestParseManifest_missingSlug(t *testing.T) {
	src := []byte(`{"version": "1.0.0", "display_name": "X", "category": "y", "description": "z"}`)
	_, err := ParseManifest(src)
	if err == nil || !strings.Contains(err.Error(), "slug") {
		t.Fatalf("expected slug error, got %v", err)
	}
}

func TestParseManifest_invalidSlugFormat(t *testing.T) {
	src := []byte(`{"slug": "Bad Slug!", "version": "1.0.0", "display_name": "X", "category": "y", "description": "z"}`)
	_, err := ParseManifest(src)
	if err == nil || !strings.Contains(err.Error(), "slug") {
		t.Fatalf("expected slug format error, got %v", err)
	}
}

func TestParseManifest_retryDefaults(t *testing.T) {
	src := []byte(`{
		"slug": "core.set",
		"version": "1.0.0",
		"display_name": "Set",
		"category": "transform",
		"description": "...",
		"retry": {"max_attempts": 3, "backoff": "exponential", "base_ms": 500}
	}`)
	m, err := ParseManifest(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if m.Retry.MaxAttempts != 3 || m.Retry.BaseMs != 500 || m.Retry.Backoff != "exponential" {
		t.Errorf("retry: %+v", m.Retry)
	}
}
