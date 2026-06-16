// Package crypto provides AES-256-GCM encryption for secrets stored in the
// credentials table. The master key is loaded once at startup from the
// FLOWGENT_CRED_KEY env var (32 raw bytes, base64-encoded).
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
)

const keyLen = 32

// Encrypt seals plaintext with AES-256-GCM. The returned ciphertext is
// nonce(12) || sealed(plaintext + 16-byte tag).
func Encrypt(plaintext, key []byte) ([]byte, error) {
	if len(key) != keyLen {
		return nil, fmt.Errorf("crypto: key must be %d bytes, got %d", keyLen, len(key))
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("crypto: new cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("crypto: new gcm: %w", err)
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("crypto: nonce: %w", err)
	}
	sealed := aead.Seal(nil, nonce, plaintext, nil)
	return append(nonce, sealed...), nil
}

// Decrypt unseals a ciphertext produced by Encrypt.
func Decrypt(ciphertext, key []byte) ([]byte, error) {
	if len(key) != keyLen {
		return nil, fmt.Errorf("crypto: key must be %d bytes, got %d", keyLen, len(key))
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("crypto: new cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("crypto: new gcm: %w", err)
	}
	if len(ciphertext) < aead.NonceSize() {
		return nil, errors.New("crypto: ciphertext too short")
	}
	nonce, sealed := ciphertext[:aead.NonceSize()], ciphertext[aead.NonceSize():]
	plaintext, err := aead.Open(nil, nonce, sealed, nil)
	if err != nil {
		return nil, fmt.Errorf("crypto: open: %w", err)
	}
	return plaintext, nil
}

// ParseMasterKey decodes a base64-encoded master key and ensures it is the
// right length. Use at startup with the FLOWGENT_CRED_KEY env value.
func ParseMasterKey(src string) ([]byte, error) {
	raw, err := base64.StdEncoding.DecodeString(src)
	if err != nil {
		return nil, fmt.Errorf("crypto: master key b64 decode: %w", err)
	}
	if len(raw) != keyLen {
		return nil, fmt.Errorf("crypto: master key must decode to %d bytes, got %d", keyLen, len(raw))
	}
	return raw, nil
}
