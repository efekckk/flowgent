package api_test

import (
	"bufio"
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
	"github.com/efekckk/flowgent/internal/idgen"
	"github.com/efekckk/flowgent/internal/registry"
	"github.com/efekckk/flowgent/internal/runlog"
	"github.com/efekckk/flowgent/internal/storage"
	"github.com/efekckk/flowgent/internal/storage/storagetest"
)

// newRunAPI mirrors newTriggerAPI but also wires up the RunLogRepo and a
// RunStore-equipped engine so /v1/runs/* paths and replay both work end
// to end.
func newRunAPI(t *testing.T) (http.Handler, *pgxpool.Pool, []*http.Cookie, string) {
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

	// Sign up and create one workflow to host the runs.
	cookies := signupForRuns(t, srv)
	wfID := createWorkflowForRuns(t, srv, cookies)
	return srv, pool, cookies, wfID
}

type nopRunSetExec struct{}

func (n *nopRunSetExec) Execute(_ context.Context, in map[string]any) (registry.ExecuteResult, error) {
	v, _ := in["values"].(map[string]any)
	if v == nil {
		v = map[string]any{}
	}
	return registry.ExecuteResult{Output: v, Port: "main"}, nil
}

// testRunStore is the minimal RunStore the engine needs during the replay
// test. It reads through the real repos so the inserted run row actually
// lands in the test database and the replay endpoint can read it back.
type testRunStore struct {
	wf   *storage.WorkflowRepo
	runs *storage.WorkflowRunRepo
}

func (s *testRunStore) LoadWorkflowForRun(ctx context.Context, workflowID string) (int, *executor.Workflow, error) {
	wf, err := s.wf.Get(ctx, workflowID)
	if err != nil {
		return 0, nil, err
	}
	ver, err := s.wf.GetVersion(ctx, wf.ID, wf.CurrentVersion)
	if err != nil {
		return 0, nil, err
	}
	var def executor.Workflow
	if err := json.Unmarshal(ver.Definition, &def); err != nil {
		return 0, nil, err
	}
	return wf.CurrentVersion, &def, nil
}

func (s *testRunStore) InsertRun(ctx context.Context, p executor.InsertRunParams) error {
	startedAt := p.StartedAt
	return s.runs.NewRun(ctx, storage.WorkflowRun{
		ID:              p.ID,
		WorkflowID:      p.WorkflowID,
		WorkflowVersion: p.WorkflowVersion,
		Status:          "running",
		TriggerKind:     p.TriggerKind,
		TriggerPayload:  p.TriggerPayload,
		ParentRunID:     p.ParentRunID,
		StartedAt:       &startedAt,
	})
}

func (s *testRunStore) PersistRun(ctx context.Context, runID string, _ *executor.Workflow, _ *executor.RunState,
	status, errMsg string, startedAt, finishedAt time.Time) error {
	return s.runs.UpdateRunStatus(ctx, runID, status, errMsg, &startedAt, &finishedAt)
}

func (s *testRunStore) GetTriggerPayload(ctx context.Context, runID string) (json.RawMessage, error) {
	run, err := s.runs.Get(ctx, runID)
	if err != nil {
		return nil, err
	}
	return run.TriggerPayload, nil
}

func signupForRuns(t *testing.T, srv http.Handler) []*http.Cookie {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"email": "runs@example.com", "password": "supersecret"})
	r := httptest.NewRequest(http.MethodPost, "/v1/auth/signup", bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, r)
	if w.Code != http.StatusCreated {
		t.Fatalf("signup: %d %s", w.Code, w.Body.String())
	}
	return w.Result().Cookies()
}

func createWorkflowForRuns(t *testing.T, srv http.Handler, cookies []*http.Cookie) string {
	t.Helper()
	body, _ := json.Marshal(map[string]any{
		"name": "runs-host",
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
	return resp.ID
}

// seedRun bypasses the engine to plant a workflow_run row directly via
// the repo, which keeps these handler-level tests focused on the HTTP
// surface area without tangling with end-to-end execution.
func seedRunRow(t *testing.T, pool *pgxpool.Pool, wfID, status string, payload string) string {
	t.Helper()
	runRepo := storage.NewWorkflowRunRepo(pool)
	id := idgen.NewRun()
	err := runRepo.NewRun(context.Background(), storage.WorkflowRun{
		ID:              id,
		WorkflowID:      wfID,
		WorkflowVersion: 1,
		Status:          status,
		TriggerKind:     "manual",
		TriggerPayload:  json.RawMessage(payload),
	})
	if err != nil {
		t.Fatalf("seed run: %v", err)
	}
	return id
}

func TestRunHandler_ListReturnsRecentFirst(t *testing.T) {
	srv, pool, cookies, wfID := newRunAPI(t)
	_ = seedRunRow(t, pool, wfID, "succeeded", `{}`)
	time.Sleep(5 * time.Millisecond)
	mid := seedRunRow(t, pool, wfID, "failed", `{}`)
	time.Sleep(5 * time.Millisecond)
	last := seedRunRow(t, pool, wfID, "succeeded", `{}`)

	r := authedRequest(t, http.MethodGet, "/v1/workflows/"+wfID+"/runs", "", cookies)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("status: %d body: %s", w.Code, w.Body.String())
	}
	var resp struct {
		Items      []map[string]any `json:"items"`
		NextCursor string           `json:"next_cursor"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if len(resp.Items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(resp.Items))
	}
	if resp.Items[0]["id"] != last {
		t.Errorf("expected newest first; got %v", resp.Items[0]["id"])
	}
	if resp.Items[1]["id"] != mid {
		t.Errorf("middle row out of order: %v", resp.Items[1]["id"])
	}
}

func TestRunHandler_ListFiltersByStatus(t *testing.T) {
	srv, pool, cookies, wfID := newRunAPI(t)
	_ = seedRunRow(t, pool, wfID, "succeeded", `{}`)
	failed := seedRunRow(t, pool, wfID, "failed", `{}`)
	_ = seedRunRow(t, pool, wfID, "succeeded", `{}`)

	r := authedRequest(t, http.MethodGet, "/v1/workflows/"+wfID+"/runs?status=failed", "", cookies)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("status: %d body: %s", w.Code, w.Body.String())
	}
	var resp struct {
		Items []map[string]any `json:"items"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if len(resp.Items) != 1 {
		t.Fatalf("expected 1 failed, got %d (%s)", len(resp.Items), w.Body.String())
	}
	if resp.Items[0]["id"] != failed {
		t.Errorf("wrong id: %v", resp.Items[0]["id"])
	}
}

func TestRunHandler_GetWithNodes(t *testing.T) {
	srv, pool, cookies, wfID := newRunAPI(t)
	runID := seedRunRow(t, pool, wfID, "succeeded", `{"a":1}`)
	repo := storage.NewWorkflowRunRepo(pool)
	_ = repo.InsertNodeRun(context.Background(), storage.NodeRun{
		ID:            idgen.NewNodeRun(),
		WorkflowRunID: runID,
		NodeID:        "a",
		Status:        "succeeded",
		Attempts:      1,
	})

	r := authedRequest(t, http.MethodGet, "/v1/runs/"+runID, "", cookies)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("status: %d body: %s", w.Code, w.Body.String())
	}
	var resp struct {
		Run   map[string]any   `json:"run"`
		Nodes []map[string]any `json:"nodes"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Run["id"] != runID {
		t.Errorf("run id: %v", resp.Run["id"])
	}
	if len(resp.Nodes) != 1 || resp.Nodes[0]["node_id"] != "a" {
		t.Errorf("nodes: %+v", resp.Nodes)
	}
}

func TestRunHandler_ReplayCreatesChildRun(t *testing.T) {
	srv, pool, cookies, wfID := newRunAPI(t)
	parent := seedRunRow(t, pool, wfID, "succeeded", `{"hello":"world"}`)

	r := authedRequest(t, http.MethodPost, "/v1/runs/"+parent+"/replay", "", cookies)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, r)
	if w.Code != http.StatusCreated {
		t.Fatalf("status: %d body: %s", w.Code, w.Body.String())
	}
	var resp struct {
		RunID string `json:"run_id"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.RunID == "" || resp.RunID == parent {
		t.Fatalf("replay id missing or same as parent: %q (parent %q)", resp.RunID, parent)
	}

	// The child row must reference parent and carry the same payload.
	got, err := storage.NewWorkflowRunRepo(pool).Get(context.Background(), resp.RunID)
	if err != nil {
		t.Fatalf("get child: %v", err)
	}
	if got.ParentRunID == nil || *got.ParentRunID != parent {
		t.Errorf("parent link: %v", got.ParentRunID)
	}
	if got.TriggerKind != "replay" {
		t.Errorf("trigger kind: %s", got.TriggerKind)
	}
	var clone map[string]any
	if err := json.Unmarshal(got.TriggerPayload, &clone); err != nil {
		t.Fatalf("payload unmarshal: %v (raw=%s)", err, string(got.TriggerPayload))
	}
	if clone["hello"] != "world" {
		t.Errorf("payload not cloned: %s", string(got.TriggerPayload))
	}
}

// newRunAPIWithStreamer mirrors newRunAPI but also installs a runlog.Streamer
// so the SSE handler is exercised end-to-end. It returns the streamer so the
// test can publish events that should reach the connected client.
func newRunAPIWithStreamer(t *testing.T) (http.Handler, *runlog.Streamer, []*http.Cookie, string) {
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

	streamer := runlog.New()

	srv := api.NewServer(api.Deps{
		Users:         storage.NewUserRepo(pool),
		Workspaces:    storage.NewWorkspaceRepo(pool),
		Sessions:      storage.NewSessionRepo(pool),
		Workflows:     wfRepo,
		Runs:          runRepo,
		RunLogs:       logRepo,
		Streamer:      streamer,
		Triggers:      storage.NewTriggerRepo(pool),
		Engine:        eng,
		Throttle:      auth.NewLoginThrottle(5, 15*time.Minute, time.Now),
		CookieDomain:  "localhost",
		CookieSecure:  false,
		PublicBaseURL: "http://flowgent.test",
	})

	cookies := signupForRuns(t, srv)
	wfID := createWorkflowForRuns(t, srv, cookies)
	return srv, streamer, cookies, wfID
}

// TestStreamRun_DeliversLiveEvents spins up a real HTTP server because the
// SSE handler relies on http.Flusher, which httptest.NewRecorder does not
// implement. The test subscribes via the SSE endpoint, publishes one event
// from a goroutine after a short delay, and asserts the message arrives in
// the response stream before the deadline.
func TestStreamRun_DeliversLiveEvents(t *testing.T) {
	srv, streamer, cookies, _ := newRunAPIWithStreamer(t)
	runID := "run_stream_test_1"

	hs := httptest.NewServer(srv)
	defer hs.Close()

	req, err := http.NewRequest(http.MethodGet, hs.URL+"/v1/runs/"+runID+"/stream", nil)
	if err != nil {
		t.Fatalf("new req: %v", err)
	}
	for _, c := range cookies {
		req.AddCookie(c)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: %d", resp.StatusCode)
	}

	// Publish from another goroutine after a small delay so the handler
	// has time to subscribe.
	go func() {
		time.Sleep(50 * time.Millisecond)
		streamer.Publish(context.Background(), runlog.Event{
			RunID:   runID,
			Message: "live-event-marker",
			Level:   "info",
		})
	}()

	done := make(chan string, 1)
	go func() {
		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.Contains(line, "live-event-marker") {
				done <- line
				return
			}
		}
		done <- ""
	}()

	select {
	case line := <-done:
		if !strings.Contains(line, "live-event-marker") {
			t.Fatalf("scanner ended without event; last line=%q", line)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("did not receive live event within 2s")
	}
}

func TestRunHandler_GetLogsReturnsAppendedRows(t *testing.T) {
	srv, pool, cookies, wfID := newRunAPI(t)
	runID := seedRunRow(t, pool, wfID, "running", `{}`)
	logRepo := storage.NewRunLogRepo(pool)
	for _, msg := range []string{"hello", "world"} {
		_ = logRepo.Append(context.Background(), storage.RunLog{
			RunID: runID, Level: "info", Message: msg,
		})
	}

	r := authedRequest(t, http.MethodGet, "/v1/runs/"+runID+"/logs", "", cookies)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("status: %d body: %s", w.Code, w.Body.String())
	}
	var resp struct {
		Items []map[string]any `json:"items"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if len(resp.Items) != 2 {
		t.Fatalf("expected 2 log rows, got %d", len(resp.Items))
	}
	if resp.Items[0]["message"] != "hello" || resp.Items[1]["message"] != "world" {
		t.Errorf("order: %+v", resp.Items)
	}
}
