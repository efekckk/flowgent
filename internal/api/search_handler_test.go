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

// newSearchAPI wires up the same Deps as the run handler tests but also
// returns the pool and the freshly-signed-up workspace id so seed
// helpers can plant log rows attached to a real workflow_run.
func newSearchAPI(t *testing.T) (http.Handler, *pgxpool.Pool, []*http.Cookie, string, string) {
	t.Helper()
	dsn := storagetest.Fresh(t)
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	t.Cleanup(pool.Close)

	reg := registry.New()
	reg.Register("core.set", &nopRunSetExec{})

	wfRepo := storage.NewWorkflowRepo(pool)
	runRepo := storage.NewWorkflowRunRepo(pool)
	logRepo := storage.NewRunLogRepo(pool)

	eng := executor.NewEngine(reg, executor.WithRunStore(&testRunStore{
		wf:   wfRepo,
		runs: runRepo,
	}))

	srv := api.NewServer(api.Deps{
		Users:         storage.NewUserRepo(pool),
		Workspaces:    storage.NewWorkspaceRepo(pool),
		Sessions:      storage.NewSessionRepo(pool),
		Workflows:     wfRepo,
		Runs:          runRepo,
		RunLogs:       logRepo,
		Triggers:      storage.NewTriggerRepo(pool),
		Engine:        eng,
		Throttle:      auth.NewLoginThrottle(5, 15*time.Minute, time.Now),
		CookieDomain:  "localhost",
		CookieSecure:  false,
		PublicBaseURL: "http://flowgent.test",
	})

	cookies, wsID := signupForSearch(t, srv)
	wfID := createWorkflowForRuns(t, srv, cookies)
	return srv, pool, cookies, wsID, wfID
}

// signupForSearch creates the user and returns the cookies and the
// workspace id baked into the signup response.
func signupForSearch(t *testing.T, srv http.Handler) ([]*http.Cookie, string) {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"email": "search@example.com", "password": "supersecret"})
	r := httptest.NewRequest(http.MethodPost, "/v1/auth/signup", bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, r)
	if w.Code != http.StatusCreated {
		t.Fatalf("signup: %d %s", w.Code, w.Body.String())
	}
	var resp struct {
		Workspace struct {
			ID string `json:"id"`
		} `json:"workspace"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Workspace.ID == "" {
		t.Fatalf("workspace id empty: %s", w.Body.String())
	}
	return w.Result().Cookies(), resp.Workspace.ID
}

// seedSearchLog plants one run_log row attributed to a real workflow_run
// inside wfID so the workspace JOIN in RunLogRepo.Search resolves.
func seedSearchLog(t *testing.T, pool *pgxpool.Pool, wfID, runID, message string) {
	t.Helper()
	runRepo := storage.NewWorkflowRunRepo(pool)
	// Idempotent: ignore conflict if the run row already exists from a
	// prior seed in the same test.
	_ = runRepo.NewRun(context.Background(), storage.WorkflowRun{
		ID:              runID,
		WorkflowID:      wfID,
		WorkflowVersion: 1,
		Status:          "succeeded",
		TriggerKind:     "manual",
		TriggerPayload:  json.RawMessage(`{}`),
	})
	logRepo := storage.NewRunLogRepo(pool)
	if err := logRepo.Append(context.Background(), storage.RunLog{
		RunID:   runID,
		Level:   "info",
		Message: message,
	}); err != nil {
		t.Fatalf("append log: %v", err)
	}
}

func TestSearch_Found(t *testing.T) {
	srv, pool, cookies, wsID, wfID := newSearchAPI(t)
	seedSearchLog(t, pool, wfID, "run_a", "slack channel notify ok")
	seedSearchLog(t, pool, wfID, "run_b", "postgres connection refused")

	r := authedRequest(t, http.MethodGet, "/v1/workspaces/"+wsID+"/runs/search?q=slack", "", cookies)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("status: %d body=%s", w.Code, w.Body.String())
	}
	var out struct {
		Hits []map[string]any `json:"hits"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	if len(out.Hits) != 1 {
		t.Fatalf("hits: %d body=%s", len(out.Hits), w.Body.String())
	}
	msg, _ := out.Hits[0]["message"].(string)
	if !strings.Contains(msg, "slack") {
		t.Errorf("hit: %+v", out.Hits[0])
	}
}

func TestSearch_QueryTooShortRejected(t *testing.T) {
	srv, _, cookies, wsID, _ := newSearchAPI(t)
	r := authedRequest(t, http.MethodGet, "/v1/workspaces/"+wsID+"/runs/search?q=ab", "", cookies)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status: %d", w.Code)
	}
}

func TestSearch_WrongWorkspaceForbidden(t *testing.T) {
	srv, _, cookies, _, _ := newSearchAPI(t)
	r := authedRequest(t, http.MethodGet, "/v1/workspaces/ws_someone_else/runs/search?q=slack", "", cookies)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, r)
	if w.Code != http.StatusForbidden {
		t.Errorf("status: %d", w.Code)
	}
}

func TestSearch_LimitCapped(t *testing.T) {
	srv, pool, cookies, wsID, wfID := newSearchAPI(t)
	for i, runID := range []string{"run_x1", "run_x2", "run_x3", "run_x4", "run_x5"} {
		_ = i
		seedSearchLog(t, pool, wfID, runID, "test message with searchterm")
	}
	r := authedRequest(t, http.MethodGet, "/v1/workspaces/"+wsID+"/runs/search?q=searchterm&limit=2", "", cookies)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("status: %d body=%s", w.Code, w.Body.String())
	}
	var out struct {
		Hits []map[string]any `json:"hits"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	if len(out.Hits) != 2 {
		t.Errorf("expected limit=2, got %d", len(out.Hits))
	}
}

func TestSearch_NoQueryRejected(t *testing.T) {
	srv, _, cookies, wsID, _ := newSearchAPI(t)
	r := authedRequest(t, http.MethodGet, "/v1/workspaces/"+wsID+"/runs/search", "", cookies)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status: %d", w.Code)
	}
}
