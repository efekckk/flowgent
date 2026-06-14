package logging

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
)

func TestLogger_redactsKnownSecretKeys(t *testing.T) {
	var buf bytes.Buffer
	lg := NewLogger(&buf, "debug")

	lg.Info("login attempt",
		"email", "user@example.com",
		"password", "supersecret",
		"password_hash", "$argon2id$h",
		"session_token", "abc.def",
		"token", "tok.xyz",
		"api_key", "sk-abc123",
		"auth_key", "key-xyz",
		"cred_payload", "encrypted-blob",
		"webhook_token", "whk_abc",
	)

	var parsed map[string]any
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("invalid json: %v\nraw: %s", err, buf.String())
	}
	if parsed["email"] != "user@example.com" {
		t.Errorf("email must not be redacted, got %v", parsed["email"])
	}
	for _, key := range []string{
		"password", "password_hash", "session_token", "token",
		"api_key", "auth_key", "cred_payload", "webhook_token",
	} {
		if parsed[key] != "[REDACTED]" {
			t.Errorf("%s must be redacted, got %v", key, parsed[key])
		}
	}
}

func TestLogger_suppressesNestedGroupUnderSensitiveKey(t *testing.T) {
	var buf bytes.Buffer
	lg := NewLogger(&buf, "debug")

	lg.Info("auth",
		slog.Group("password",
			"hash", "bcrypt$2a$10$realHash",
			"algo", "bcrypt",
		),
	)

	if strings.Contains(buf.String(), "realHash") || strings.Contains(buf.String(), "bcrypt") {
		t.Errorf("sensitive group children must not appear in output, got %s", buf.String())
	}
}

func TestLogger_redactsBearerInAuthorizationHeader(t *testing.T) {
	var buf bytes.Buffer
	lg := NewLogger(&buf, "debug")
	lg.Info("inbound", "authorization", "Bearer pat_abc123")

	if !strings.Contains(buf.String(), `"authorization":"[REDACTED]"`) {
		t.Errorf("authorization header must be redacted, got %s", buf.String())
	}
}

func TestLogger_levelFilter(t *testing.T) {
	var buf bytes.Buffer
	lg := NewLogger(&buf, "warn")
	lg.Info("noise")
	if buf.Len() != 0 {
		t.Errorf("info must be filtered under warn, got %s", buf.String())
	}
	lg.Warn("real")
	if !strings.Contains(buf.String(), "real") {
		t.Errorf("warn must pass through, got %s", buf.String())
	}
}
