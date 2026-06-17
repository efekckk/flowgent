package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// signupAs creates a fresh user/workspace + session and returns the cookie
// jar. Each call uses a unique email so two callers within the same test
// produce two independent tenants.
func signupAs(t *testing.T, srv http.Handler, email, password string) []*http.Cookie {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"email": email, "password": password})
	r := httptest.NewRequest(http.MethodPost, "/v1/auth/signup", bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, r)
	if w.Code != http.StatusCreated {
		t.Fatalf("signup %s: %d %s", email, w.Code, w.Body.String())
	}
	return w.Result().Cookies()
}

// createWorkflowAs is the multi-tenant version of createWorkflowForRuns.
func createWorkflowAs(t *testing.T, srv http.Handler, cookies []*http.Cookie, name string) string {
	t.Helper()
	body, _ := json.Marshal(map[string]any{
		"name": name,
		"definition": map[string]any{
			"nodes": []any{
				map[string]any{"id": "a", "tool": "core.set", "params": map[string]any{"values": map[string]any{"x": 1}}},
			},
			"edges": []any{},
		},
	})
	r := httptest.NewRequest(http.MethodPost, "/v1/workflows", bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	for _, c := range cookies {
		r.AddCookie(c)
	}
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, r)
	if w.Code != http.StatusCreated {
		t.Fatalf("create workflow: %d %s", w.Code, w.Body.String())
	}
	var resp struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.ID == "" {
		t.Fatalf("empty wf id: %s", w.Body.String())
	}
	return resp.ID
}

// TestOwnership_CrossTenantAccessReturns404 confirms that a session for
// user B cannot reach any of user A's workflow-, run-, or trigger-scoped
// endpoints. Cross-tenant reads must be indistinguishable from "does not
// exist" so resource existence in another workspace stays hidden.
//
// The single end-to-end check here is intentionally broad rather than a
// per-handler suite: every handler funnels through the loadOwned* helpers
// in internal/api/ownership.go, so a passing matrix here is the canonical
// proof that the guard is wired in across the surface area. Per-handler
// happy paths are covered by the existing run_handler_test /
// trigger_handler_test / workflow_handler_test suites.
func TestOwnership_CrossTenantAccessReturns404(t *testing.T) {
	srv, pool, ownerCookies, _ := newRunAPI(t)

	// Owner already has a workflow created by newRunAPI; seed a run on it.
	// newRunAPI's createWorkflowForRuns returns the wf id via the response,
	// but the helper doesn't surface it back. Re-create one we control.
	ownerWF := createWorkflowAs(t, srv, ownerCookies, "owner-wf")
	ownerRun := seedRunRow(t, pool, ownerWF, "succeeded", `{}`)

	// Intruder signs up in the same server → gets their own workspace.
	intruder := signupAs(t, srv, "intruder@example.com", "supersecret")

	cases := []struct {
		name   string
		method string
		path   string
	}{
		{"get workflow", http.MethodGet, "/v1/workflows/" + ownerWF},
		{"run workflow", http.MethodPost, "/v1/workflows/" + ownerWF + "/run"},
		{"chat workflow", http.MethodPost, "/v1/workflows/" + ownerWF + "/chat"},
		{"list runs", http.MethodGet, "/v1/workflows/" + ownerWF + "/runs"},
		{"list triggers", http.MethodGet, "/v1/workflows/" + ownerWF + "/triggers"},
		{"create trigger", http.MethodPost, "/v1/workflows/" + ownerWF + "/triggers"},
		{"get run", http.MethodGet, "/v1/runs/" + ownerRun},
		{"run logs", http.MethodGet, "/v1/runs/" + ownerRun + "/logs"},
		{"run stream", http.MethodGet, "/v1/runs/" + ownerRun + "/stream"},
		{"replay run", http.MethodPost, "/v1/runs/" + ownerRun + "/replay"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := authedRequest(t, tc.method, tc.path, `{}`, intruder)
			w := httptest.NewRecorder()
			srv.ServeHTTP(w, r)
			// 404 is the canonical "doesn't exist" response; some endpoints
			// can also legitimately return 503 when a sub-dependency
			// (RunLogs) is disabled. Either way the intruder must NOT
			// receive 200/201 or any data leak.
			if w.Code == http.StatusOK || w.Code == http.StatusCreated {
				t.Fatalf("intruder reached %s %s: status %d body %s",
					tc.method, tc.path, w.Code, w.Body.String())
			}
			if w.Code != http.StatusNotFound && w.Code != http.StatusServiceUnavailable {
				// A 400/401/403 isn't a security bug but signals the guard
				// is unevenly applied. Worth surfacing as a soft failure.
				t.Errorf("%s %s returned %d (want 404), body=%s",
					tc.method, tc.path, w.Code, w.Body.String())
			}
		})
	}
}
