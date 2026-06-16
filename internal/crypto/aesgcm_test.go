package crypto

import (
	"bytes"
	"encoding/base64"
	"testing"
)

var testKey = func() []byte {
	k := make([]byte, 32)
	for i := range k {
		k[i] = byte(i)
	}
	return k
}()

func TestEncryptDecrypt_roundTrip(t *testing.T) {
	plain := []byte("supersecret-api-key-sk-xxxxxx")
	ct, err := Encrypt(plain, testKey)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	if bytes.Contains(ct, plain) {
		t.Errorf("ciphertext leaks plaintext")
	}
	pt, err := Decrypt(ct, testKey)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if !bytes.Equal(pt, plain) {
		t.Errorf("round trip lost data: %q", pt)
	}
}

func TestEncrypt_uniqueNoncePerCall(t *testing.T) {
	plain := []byte("same input")
	a, _ := Encrypt(plain, testKey)
	b, _ := Encrypt(plain, testKey)
	if bytes.Equal(a, b) {
		t.Errorf("two encryptions must differ (nonce randomness)")
	}
}

func TestDecrypt_wrongKeyFails(t *testing.T) {
	plain := []byte("x")
	ct, _ := Encrypt(plain, testKey)

	wrong := make([]byte, 32)
	for i := range wrong {
		wrong[i] = byte(i + 1)
	}
	_, err := Decrypt(ct, wrong)
	if err == nil {
		t.Fatalf("expected error with wrong key")
	}
}

func TestDecrypt_tooShortFails(t *testing.T) {
	_, err := Decrypt([]byte{1, 2, 3}, testKey)
	if err == nil {
		t.Fatalf("expected error on too-short ciphertext")
	}
}

func TestParseMasterKey_validBase64(t *testing.T) {
	src := base64.StdEncoding.EncodeToString(testKey)
	k, err := ParseMasterKey(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if !bytes.Equal(k, testKey) {
		t.Errorf("key round trip failed")
	}
}

func TestParseMasterKey_wrongLength(t *testing.T) {
	short := base64.StdEncoding.EncodeToString([]byte("only-15-chars-x"))
	_, err := ParseMasterKey(short)
	if err == nil {
		t.Fatalf("expected error for 15-byte key")
	}
}
