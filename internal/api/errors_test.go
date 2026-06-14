package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWriteJSON_setsHeaderAndBody(t *testing.T) {
	rr := httptest.NewRecorder()
	WriteJSON(rr, http.StatusOK, map[string]string{"hello": "world"})
	if got := rr.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("want application/json, got %q", got)
	}
	var body map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if body["hello"] != "world" {
		t.Errorf("body mismatch: %v", body)
	}
}

func TestWriteError_envelope(t *testing.T) {
	rr := httptest.NewRecorder()
	WriteError(rr, http.StatusUnauthorized, "invalid_credentials", "Email or password is wrong.")
	var env ErrorEnvelope
	if err := json.Unmarshal(rr.Body.Bytes(), &env); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if env.Error.Code != "invalid_credentials" {
		t.Errorf("code: %+v", env)
	}
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("status: %d", rr.Code)
	}
}
