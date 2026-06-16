// Package webhook serves inbound HTTP triggers. Each webhook trigger is
// addressable at /webhooks/{trigger_id}/{token}; when a secret is configured
// on the trigger, requests must also carry an X-Flowgent-Signature header
// containing `sha256=<hex>` computed over the raw request body. Verification
// is constant-time to prevent timing attacks.
package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// VerifySignature returns true iff `signatureHeader` matches HMAC-SHA256(body,
// secret) using a constant-time comparison. The header format is
// "sha256=<hex>"; case-insensitive on the prefix. An empty secret never
// matches — callers should skip verification entirely when no secret is set.
func VerifySignature(body, secret []byte, signatureHeader string) bool {
	if len(secret) == 0 {
		return false
	}
	const prefix = "sha256="
	if len(signatureHeader) <= len(prefix) {
		return false
	}
	if !strings.EqualFold(signatureHeader[:len(prefix)], prefix) {
		return false
	}
	want, err := hex.DecodeString(signatureHeader[len(prefix):])
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, secret)
	mac.Write(body)
	return hmac.Equal(mac.Sum(nil), want)
}
