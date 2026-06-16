package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/efekckk/flowgent/internal/api"
	"github.com/efekckk/flowgent/internal/auth"
	"github.com/efekckk/flowgent/internal/executor"
	"github.com/efekckk/flowgent/internal/registry"
	"github.com/efekckk/flowgent/internal/storage"
	"github.com/efekckk/flowgent/internal/storage/storagetest"
)

func newTriggerAPI(t *testing.T) (http.Handler, *pgxpool.Pool) {
	t.Helper()
	dsn := storagetest.Fresh(t)
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	t.Cleanup(pool.Close)

	reg := registry.New()
	reg.Register("core.set", &nopTriggerSetExec{})

	srv := api.NewServer(api.Deps{
		Users:         storage.NewUserRepo(pool),
		Workspaces:    storage.NewWorkspaceRepo(pool),
		Sessions:      storage.NewSessionRepo(pool),
		Workflows:     storage.NewWorkflowRepo(pool),
		Runs:          storage.NewWorkflowRunRepo(pool),
		Triggers:      storage.NewTriggerRepo(pool),
		Engine:        executor.NewEngine(reg),
		Throttle:      auth.NewLoginThrottle(5, 15*time.Minute, time.Now),
		CookieDomain:  "localhost",
		CookieSecure:  false,
		PublicBaseURL: "http://flowgent.test",
	})
	return srv, pool
}

type nopTriggerSetExec struct{}

func (n *nopTriggerSetExec) Execute(_ context.Context, in map[string]any) (registry.ExecuteResult, error) {
	v, _ := in["values"].(map[string]any)
	if v == nil {
		v = map[string]any{}
	}
	return registry.ExecuteResult{Output: v, Port: "main"}, nil
}

func signupForTriggers(t *testing.T, srv http.Handler) []*http.Cookie {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"email": "trg@example.com", "password": "supersecret"})
	r := httptest.NewRequest(http.MethodPost, "/v1/auth/signup", bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, r)
	if w.Code != http.StatusCreated {
		t.Fatalf("signup: %d %s", w.Code, w.Body.String())
	}
	return w.Result().Cookies()
}

func createWorkflowForTriggers(t *testing.T, srv http.Handler, cookies []*http.Cookie) string {
	t.Helper()
	body, _ := json.Marshal(map[string]any{
		"name": "trigger-host",
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
		t.Fatalf("workflow id empty: %s", w.Body.String())
	}
	return resp.ID
}

func setupTriggerCtx(t *testing.T) (http.Handler, []*http.Cookie, string) {
	t.Helper()
	srv, _ := newTriggerAPI(t)
	cookies := signupForTriggers(t, srv)
	wfID := createWorkflowForTriggers(t, srv, cookies)
	return srv, cookies, wfID
}

func authedRequest(t *testing.T, method, path string, body string, cookies []*http.Cookie) *http.Request {
	t.Helper()
	var reader *bytes.Reader
	if body != "" {
		reader = bytes.NewReader([]byte(body))
	}
	var r *http.Request
	if reader != nil {
		r = httptest.NewRequest(method, path, reader)
		r.Header.Set("Content-Type", "application/json")
	} else {
		r = httptest.NewRequest(method, path, nil)
	}
	for _, c := range cookies {
		r.AddCookie(c)
	}
	return r
}

func postTrigger(t *testing.T, srv http.Handler, cookies []*http.Cookie, wfID, body string) map[string]any {
	t.Helper()
	r := authedRequest(t, http.MethodPost, "/v1/workflows/"+wfID+"/triggers", body, cookies)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, r)
	if w.Code != http.StatusCreated {
		t.Fatalf("postTrigger: %d body=%s", w.Code, w.Body.String())
	}
	var out map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	return out
}

func TestTriggers_CreateCron(t *testing.T) {
	srv, cookies, wfID := setupTriggerCtx(t)

	r := authedRequest(t, http.MethodPost, "/v1/workflows/"+wfID+"/triggers",
		`{"kind":"cron","config":{"cron":"@every 5m"}}`, cookies)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, r)
	if w.Code != http.StatusCreated {
		t.Fatalf("status: %d body=%s", w.Code, w.Body.String())
	}
	var out struct {
		ID      string         `json:"id"`
		Kind    string         `json:"kind"`
		Enabled bool           `json:"enabled"`
		Config  map[string]any `json:"config"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out.Kind != "cron" || out.ID == "" || !out.Enabled {
		t.Errorf("out: %+v", out)
	}
	if out.Config["cron"] != "@every 5m" {
		t.Errorf("config: %+v", out.Config)
	}
}

func TestTriggers_CreateWebhookReturnsURL(t *testing.T) {
	srv, cookies, wfID := setupTriggerCtx(t)

	r := authedRequest(t, http.MethodPost, "/v1/workflows/"+wfID+"/triggers",
		`{"kind":"webhook","config":{}}`, cookies)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, r)
	if w.Code != http.StatusCreated {
		t.Fatalf("status: %d body=%s", w.Code, w.Body.String())
	}
	var out struct {
		ID         string         `json:"id"`
		WebhookURL string         `json:"webhook_url"`
		Config     map[string]any `json:"config"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out.WebhookURL == "" {
		t.Fatalf("webhook_url not returned: %s", w.Body.String())
	}
	if !strings.Contains(out.WebhookURL, out.ID) {
		t.Errorf("webhook_url should include trigger id: %s", out.WebhookURL)
	}
	tok, _ := out.Config["token"].(string)
	if tok == "" {
		t.Errorf("expected auto-generated token in config: %+v", out.Config)
	}
	if !strings.Contains(out.WebhookURL, tok) {
		t.Errorf("webhook_url should include token: %s", out.WebhookURL)
	}
}

func TestTriggers_InvalidCronRejected(t *testing.T) {
	srv, cookies, wfID := setupTriggerCtx(t)

	r := authedRequest(t, http.MethodPost, "/v1/workflows/"+wfID+"/triggers",
		`{"kind":"cron","config":{"cron":"not a cron"}}`, cookies)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestTriggers_UnknownKindRejected(t *testing.T) {
	srv, cookies, wfID := setupTriggerCtx(t)

	r := authedRequest(t, http.MethodPost, "/v1/workflows/"+wfID+"/triggers",
		`{"kind":"queue","config":{}}`, cookies)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestTriggers_List(t *testing.T) {
	srv, cookies, wfID := setupTriggerCtx(t)
	postTrigger(t, srv, cookies, wfID, `{"kind":"cron","config":{"cron":"@daily"}}`)
	postTrigger(t, srv, cookies, wfID, `{"kind":"webhook","config":{}}`)

	r := authedRequest(t, http.MethodGet, "/v1/workflows/"+wfID+"/triggers", "", cookies)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("status: %d", w.Code)
	}
	var out struct {
		Items []map[string]any `json:"items"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	if len(out.Items) != 2 {
		t.Fatalf("len: %d body=%s", len(out.Items), w.Body.String())
	}
	// The webhook entry must carry a webhook_url.
	foundWebhookURL := false
	for _, item := range out.Items {
		if item["kind"] == "webhook" {
			if u, _ := item["webhook_url"].(string); u != "" {
				foundWebhookURL = true
			}
		}
	}
	if !foundWebhookURL {
		t.Errorf("expected webhook_url on listed webhook trigger: %s", w.Body.String())
	}
}

func TestTriggers_PatchToggleEnabled(t *testing.T) {
	srv, cookies, wfID := setupTriggerCtx(t)
	created := postTrigger(t, srv, cookies, wfID, `{"kind":"cron","config":{"cron":"@hourly"}}`)
	id, _ := created["id"].(string)
	if id == "" {
		t.Fatalf("no id in created: %+v", created)
	}

	r := authedRequest(t, http.MethodPatch, "/v1/triggers/"+id,
		`{"enabled":false,"config":{"cron":"@hourly"}}`, cookies)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("status: %d body=%s", w.Code, w.Body.String())
	}
	var out struct {
		Enabled bool `json:"enabled"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	if out.Enabled {
		t.Errorf("expected enabled=false, got true: %s", w.Body.String())
	}
}

func TestTriggers_Delete(t *testing.T) {
	srv, cookies, wfID := setupTriggerCtx(t)
	created := postTrigger(t, srv, cookies, wfID, `{"kind":"webhook","config":{}}`)
	id, _ := created["id"].(string)
	if id == "" {
		t.Fatalf("no id in created: %+v", created)
	}

	r := authedRequest(t, http.MethodDelete, "/v1/triggers/"+id, "", cookies)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, r)
	if w.Code != http.StatusNoContent {
		t.Errorf("status: %d body=%s", w.Code, w.Body.String())
	}
}
