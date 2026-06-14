package auth

import (
	"strings"
	"testing"
)

func TestGenerateSessionToken_format(t *testing.T) {
	tok, err := GenerateSessionToken()
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if !strings.HasPrefix(tok, "fg_") {
		t.Errorf("expected fg_ prefix, got %q", tok)
	}
	if len(tok) < 30 {
		t.Errorf("token too short: %d chars", len(tok))
	}
}

func TestHashSessionToken_deterministic(t *testing.T) {
	tok := "fg_abc"
	a := HashSessionToken(tok)
	b := HashSessionToken(tok)
	if len(a) != 32 {
		t.Errorf("expected 32-byte sha256, got %d", len(a))
	}
	if string(a) != string(b) {
		t.Errorf("hash must be deterministic")
	}
}

func TestGenerateSessionToken_unique(t *testing.T) {
	seen := make(map[string]struct{}, 1000)
	for i := 0; i < 1000; i++ {
		tok, _ := GenerateSessionToken()
		if _, dup := seen[tok]; dup {
			t.Fatalf("duplicate token at i=%d", i)
		}
		seen[tok] = struct{}{}
	}
}
