package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/efekckk/flowgent/internal/api"
)

func TestHealthEndpoint_returnsOK(t *testing.T) {
	srv := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()
	srv.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), `"status":"ok"`) {
		t.Fatalf("expected status ok body, got %q", rr.Body.String())
	}
}

func TestEnvOr_returnsFallbackWhenUnset(t *testing.T) {
	t.Setenv("FLG_TEST_UNSET", "")
	if got := envOr("FLG_TEST_UNSET", "default"); got != "default" {
		t.Fatalf("got %q", got)
	}
}

func TestEnvOr_returnsValueWhenSet(t *testing.T) {
	t.Setenv("FLG_TEST_SET", "live")
	if got := envOr("FLG_TEST_SET", "default"); got != "live" {
		t.Fatalf("got %q", got)
	}
}

func newTestServer(t *testing.T) http.Handler {
	t.Helper()
	return newSmokeServer(t)
}

// newSmokeServer builds a router with empty Deps for tests that only need
// the unauthenticated routes (e.g., /health).
func newSmokeServer(t *testing.T) http.Handler {
	t.Helper()
	return api.NewServer(api.Deps{})
}
