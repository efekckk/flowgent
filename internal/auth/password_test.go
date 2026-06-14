package auth

import (
	"strings"
	"testing"
)

func TestHashPassword_producesArgon2idEncodedHash(t *testing.T) {
	h, err := HashPassword("hunter2")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	if !strings.HasPrefix(h, "$argon2id$") {
		t.Errorf("expected $argon2id$ prefix, got %q", h)
	}
}

func TestVerifyPassword_acceptsCorrectPassword(t *testing.T) {
	h, _ := HashPassword("hunter2")
	ok, err := VerifyPassword(h, "hunter2")
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if !ok {
		t.Fatalf("verify should accept correct password")
	}
}

func TestVerifyPassword_rejectsWrongPassword(t *testing.T) {
	h, _ := HashPassword("hunter2")
	ok, _ := VerifyPassword(h, "wrong")
	if ok {
		t.Fatalf("verify should reject wrong password")
	}
}

func TestHashPassword_differentSaltEachTime(t *testing.T) {
	a, _ := HashPassword("same")
	b, _ := HashPassword("same")
	if a == b {
		t.Fatalf("expected different hashes due to salt")
	}
}

func TestVerifyPassword_malformedRejected(t *testing.T) {
	if _, err := VerifyPassword("not-a-hash", "x"); err == nil {
		t.Fatalf("expected error for malformed hash")
	}
}
