package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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

func newWFAPI(t *testing.T) (http.Handler, *pgxpool.Pool, *registry.Registry) {
	t.Helper()
	dsn := storagetest.Fresh(t)
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	t.Cleanup(pool.Close)

	reg := registry.New()
	reg.Register("core.set", &nopSetExec{})

	srv := api.NewServer(api.Deps{
		Users:        storage.NewUserRepo(pool),
		Workspaces:   storage.NewWorkspaceRepo(pool),
		Sessions:     storage.NewSessionRepo(pool),
		Workflows:    storage.NewWorkflowRepo(pool),
		Runs:         storage.NewWorkflowRunRepo(pool),
		Engine:       executor.NewEngine(reg),
		Throttle:     auth.NewLoginThrottle(5, 15*time.Minute, time.Now),
		CookieDomain: "localhost",
		CookieSecure: false,
	})
	return srv, pool, reg
}

type nopSetExec struct{}

func (n *nopSetExec) Execute(_ context.Context, in map[string]any) (registry.ExecuteResult, error) {
	v, _ := in["values"].(map[string]any)
	if v == nil {
		v = map[string]any{}
	}
	return registry.ExecuteResult{Output: v, Port: "main"}, nil
}

func signupForWorkflow(t *testing.T, srv http.Handler) []*http.Cookie {
	body, _ := json.Marshal(map[string]string{"email": "wf@example.com", "password": "supersecret"})
	r := httptest.NewRequest(http.MethodPost, "/v1/auth/signup", bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, r)
	if w.Code != http.StatusCreated {
		t.Fatalf("signup: %d %s", w.Code, w.Body.String())
	}
	return w.Result().Cookies()
}

func TestCreateWorkflow_returnsDraftWithVersion1(t *testing.T) {
	srv, _, _ := newWFAPI(t)
	cookies := signupForWorkflow(t, srv)

	body, _ := json.Marshal(map[string]any{
		"name": "demo",
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
		t.Fatalf("status: %d body: %s", w.Code, w.Body.String())
	}
	var resp struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Status != "draft" {
		t.Errorf("status: %s", resp.Status)
	}
	if resp.ID == "" {
		t.Errorf("missing id")
	}
}

func TestCreateWorkflow_requiresAuth(t *testing.T) {
	srv, _, _ := newWFAPI(t)
	body, _ := json.Marshal(map[string]any{"name": "x", "definition": map[string]any{}})
	r := httptest.NewRequest(http.MethodPost, "/v1/workflows", bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status: %d", w.Code)
	}
}

func TestRunWorkflow_executesAndPersists(t *testing.T) {
	srv, pool, _ := newWFAPI(t)
	cookies := signupForWorkflow(t, srv)

	createBody, _ := json.Marshal(map[string]any{
		"name": "demo",
		"definition": map[string]any{
			"nodes": []any{
				map[string]any{"id": "a", "tool": "core.set", "params": map[string]any{"values": map[string]any{"x": 1}}},
			},
			"edges": []any{},
		},
	})
	r := httptest.NewRequest(http.MethodPost, "/v1/workflows", bytes.NewReader(createBody))
	r.Header.Set("Content-Type", "application/json")
	for _, c := range cookies {
		r.AddCookie(c)
	}
	rr := httptest.NewRecorder()
	srv.ServeHTTP(rr, r)
	var created struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(rr.Body.Bytes(), &created)

	runBody, _ := json.Marshal(map[string]any{})
	r2 := httptest.NewRequest(http.MethodPost, "/v1/workflows/"+created.ID+"/run", bytes.NewReader(runBody))
	r2.Header.Set("Content-Type", "application/json")
	for _, c := range cookies {
		r2.AddCookie(c)
	}
	w2 := httptest.NewRecorder()
	srv.ServeHTTP(w2, r2)
	if w2.Code != http.StatusOK {
		t.Fatalf("run status: %d body: %s", w2.Code, w2.Body.String())
	}
	var runResp struct {
		RunID  string `json:"run_id"`
		Status string `json:"status"`
	}
	_ = json.Unmarshal(w2.Body.Bytes(), &runResp)
	if runResp.Status != "succeeded" {
		t.Errorf("status: %s", runResp.Status)
	}

	var cnt int
	_ = pool.QueryRow(context.Background(),
		`SELECT count(*) FROM node_runs WHERE workflow_run_id = $1`, runResp.RunID).Scan(&cnt)
	if cnt != 1 {
		t.Errorf("node_runs: %d", cnt)
	}
}

func TestGetWorkflow_returnsCurrentDefinition(t *testing.T) {
	srv, _, _ := newWFAPI(t)
	cookies := signupForWorkflow(t, srv)
	createBody, _ := json.Marshal(map[string]any{
		"name": "gw",
		"definition": map[string]any{
			"nodes": []any{
				map[string]any{"id": "a", "tool": "core.set", "params": map[string]any{"values": map[string]any{"x": 1}}},
			},
			"edges": []any{},
		},
	})
	r := httptest.NewRequest(http.MethodPost, "/v1/workflows", bytes.NewReader(createBody))
	r.Header.Set("Content-Type", "application/json")
	for _, c := range cookies {
		r.AddCookie(c)
	}
	cw := httptest.NewRecorder()
	srv.ServeHTTP(cw, r)
	var created struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(cw.Body.Bytes(), &created)

	r2 := httptest.NewRequest(http.MethodGet, "/v1/workflows/"+created.ID, nil)
	for _, c := range cookies {
		r2.AddCookie(c)
	}
	w2 := httptest.NewRecorder()
	srv.ServeHTTP(w2, r2)
	if w2.Code != http.StatusOK {
		t.Fatalf("status: %d body: %s", w2.Code, w2.Body.String())
	}
	if !bytes.Contains(w2.Body.Bytes(), []byte(`"name":"gw"`)) {
		t.Errorf("name missing: %s", w2.Body.String())
	}
}
