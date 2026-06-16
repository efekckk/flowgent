package webhook_test

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/efekckk/flowgent/internal/webhook"
)

func TestVerifySignature_HappyPath(t *testing.T) {
	body := []byte(`{"hello":"world"}`)
	secret := []byte("super-secret")
	mac := hmac.New(sha256.New, secret)
	mac.Write(body)
	sig := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	if !webhook.VerifySignature(body, secret, sig) {
		t.Fatalf("expected verify ok")
	}
}

func TestVerifySignature_WrongSecretFails(t *testing.T) {
	body := []byte(`x`)
	mac := hmac.New(sha256.New, []byte("a"))
	mac.Write(body)
	sig := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	if webhook.VerifySignature(body, []byte("b"), sig) {
		t.Errorf("verify should fail with wrong secret")
	}
}

func TestVerifySignature_EmptySecretFails(t *testing.T) {
	if webhook.VerifySignature([]byte("x"), nil, "sha256=abc") {
		t.Errorf("empty secret must never match")
	}
}

func TestVerifySignature_MalformedHeader(t *testing.T) {
	if webhook.VerifySignature([]byte("x"), []byte("s"), "md5=abc") {
		t.Errorf("non-sha256 prefix should fail")
	}
	if webhook.VerifySignature([]byte("x"), []byte("s"), "sha256=zzz") {
		t.Errorf("non-hex value should fail")
	}
	if webhook.VerifySignature([]byte("x"), []byte("s"), "") {
		t.Errorf("empty header should fail")
	}
	if webhook.VerifySignature([]byte("x"), []byte("s"), "sha256=") {
		t.Errorf("prefix-only should fail")
	}
}

func TestVerifySignature_CaseInsensitivePrefix(t *testing.T) {
	body := []byte(`x`)
	secret := []byte("s")
	mac := hmac.New(sha256.New, secret)
	mac.Write(body)
	sig := "SHA256=" + hex.EncodeToString(mac.Sum(nil))
	if !webhook.VerifySignature(body, secret, sig) {
		t.Errorf("case-insensitive prefix should match")
	}
}

func TestHandler_UnknownTriggerReturns404(t *testing.T) {
	h := webhook.NewHandler(&fakeResolver{}, &fakeFirer{})
	req := httptest.NewRequest("POST", "/webhooks/unknown/tok", bytes.NewReader(nil))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Errorf("status: %d", rr.Code)
	}
}

func TestHandler_WrongTokenReturns404(t *testing.T) {
	res := &fakeResolver{triggers: map[string]webhook.WebhookTrigger{
		"trg": {ID: "trg", WorkflowID: "wf", Token: "real-tok"},
	}}
	h := webhook.NewHandler(res, &fakeFirer{})
	req := httptest.NewRequest("POST", "/webhooks/trg/wrong-tok", bytes.NewReader(nil))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Errorf("status: %d", rr.Code)
	}
}

func TestHandler_BadPathReturns404(t *testing.T) {
	h := webhook.NewHandler(&fakeResolver{}, &fakeFirer{})
	// missing token segment
	req := httptest.NewRequest("POST", "/webhooks/just-id", bytes.NewReader(nil))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Errorf("status: %d", rr.Code)
	}
}

func TestHandler_NoSecretAcceptsAnyJSON(t *testing.T) {
	res := &fakeResolver{triggers: map[string]webhook.WebhookTrigger{
		"trg": {ID: "trg", WorkflowID: "wf", Token: "tok"},
	}}
	firer := &fakeFirer{}
	h := webhook.NewHandler(res, firer)
	req := httptest.NewRequest("POST", "/webhooks/trg/tok", bytes.NewReader([]byte(`{"foo":"bar"}`)))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusAccepted {
		t.Errorf("status: %d body=%s", rr.Code, rr.Body.String())
	}
	if firer.payload["foo"] != "bar" {
		t.Errorf("payload not forwarded: %+v", firer.payload)
	}
	if firer.triggerID != "trg" || firer.workflowID != "wf" {
		t.Errorf("identifiers: %s/%s", firer.triggerID, firer.workflowID)
	}
}

func TestHandler_SecretRequiresHMAC(t *testing.T) {
	body := []byte(`{"x":1}`)
	secret := []byte("s3cr3t")
	mac := hmac.New(sha256.New, secret)
	mac.Write(body)
	good := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	res := &fakeResolver{triggers: map[string]webhook.WebhookTrigger{
		"trg": {ID: "trg", WorkflowID: "wf", Token: "tok", Secret: secret},
	}}
	h := webhook.NewHandler(res, &fakeFirer{})

	// missing sig -> 401
	req := httptest.NewRequest("POST", "/webhooks/trg/tok", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("missing sig status: %d", rr.Code)
	}

	// wrong sig -> 401
	req = httptest.NewRequest("POST", "/webhooks/trg/tok", bytes.NewReader(body))
	req.Header.Set("X-Flowgent-Signature", "sha256=deadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef")
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("wrong sig status: %d", rr.Code)
	}

	// correct sig -> 202
	req = httptest.NewRequest("POST", "/webhooks/trg/tok", bytes.NewReader(body))
	req.Header.Set("X-Flowgent-Signature", good)
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusAccepted {
		t.Errorf("good sig status: %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandler_NonJSONBodyForwardedRaw(t *testing.T) {
	res := &fakeResolver{triggers: map[string]webhook.WebhookTrigger{
		"trg": {ID: "trg", WorkflowID: "wf", Token: "tok"},
	}}
	firer := &fakeFirer{}
	h := webhook.NewHandler(res, firer)
	req := httptest.NewRequest("POST", "/webhooks/trg/tok", bytes.NewReader([]byte("not json")))
	req.Header.Set("Content-Type", "text/plain")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusAccepted {
		t.Errorf("status: %d", rr.Code)
	}
	if firer.payload["body"] != "not json" {
		t.Errorf("expected raw body wrapped, got %+v", firer.payload)
	}
}

func TestHandler_PayloadTooLargeReturns413(t *testing.T) {
	res := &fakeResolver{triggers: map[string]webhook.WebhookTrigger{
		"trg": {ID: "trg", WorkflowID: "wf", Token: "tok"},
	}}
	h := webhook.NewHandler(res, &fakeFirer{})
	huge := bytes.Repeat([]byte("x"), 2<<20) // 2 MiB > 1 MiB cap
	req := httptest.NewRequest("POST", "/webhooks/trg/tok", bytes.NewReader(huge))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("status: %d", rr.Code)
	}
}

func TestHandler_FirerErrorReturns500(t *testing.T) {
	res := &fakeResolver{triggers: map[string]webhook.WebhookTrigger{
		"trg": {ID: "trg", WorkflowID: "wf", Token: "tok"},
	}}
	firer := &fakeFirer{err: io.ErrUnexpectedEOF}
	h := webhook.NewHandler(res, firer)
	req := httptest.NewRequest("POST", "/webhooks/trg/tok", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("status: %d", rr.Code)
	}
}

type fakeResolver struct{ triggers map[string]webhook.WebhookTrigger }

func (f *fakeResolver) ResolveWebhook(_ context.Context, id string) (webhook.WebhookTrigger, bool, error) {
	t, ok := f.triggers[id]
	return t, ok, nil
}

type fakeFirer struct {
	triggerID  string
	workflowID string
	payload    map[string]any
	err        error
}

func (f *fakeFirer) FireTrigger(_ context.Context, triggerID, workflowID string, payload map[string]any) error {
	f.triggerID = triggerID
	f.workflowID = workflowID
	f.payload = payload
	return f.err
}
