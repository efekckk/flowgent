package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
)

const (
	sessionTokenPrefix    = "fg_"
	sessionTokenRandBytes = 32
)

// GenerateSessionToken returns a high-entropy bearer token. Only the SHA-256
// of the token is persisted; the plaintext leaves the server only once, in
// the Set-Cookie response.
func GenerateSessionToken() (string, error) {
	b := make([]byte, sessionTokenRandBytes)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("auth: session token: %w", err)
	}
	return sessionTokenPrefix + base64.RawURLEncoding.EncodeToString(b), nil
}

func HashSessionToken(token string) []byte {
	h := sha256.Sum256([]byte(token))
	return h[:]
}
