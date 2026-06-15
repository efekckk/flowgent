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

	"github.com/efekckk/flowgent/internal/agent"
	"github.com/efekckk/flowgent/internal/api"
	"github.com/efekckk/flowgent/internal/auth"
	"github.com/efekckk/flowgent/internal/executor"
	"github.com/efekckk/flowgent/internal/provider"
	"github.com/efekckk/flowgent/internal/registry"
	"github.com/efekckk/flowgent/internal/storage"
	"github.com/efekckk/flowgent/internal/storage/storagetest"
)

func newChatAPI(t *testing.T, mock *provider.Mock) (http.Handler, *pgxpool.Pool, string) {
	t.Helper()
	dsn := storagetest.Fresh(t)
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	t.Cleanup(pool.Close)

	reg := registry.New()
	reg.Register("core.set", &chatNopSetExec{})
	known := map[string]struct{}{"core.set": {}}

	ag := agent.New(agent.Deps{
		Provider:   mock,
		KnownTools: known,
		MaxRetries: 3,
	})

	srv := api.NewServer(api.Deps{
		Users:        storage.NewUserRepo(pool),
		Workspaces:   storage.NewWorkspaceRepo(pool),
		Sessions:     storage.NewSessionRepo(pool),
		Workflows:    storage.NewWorkflowRepo(pool),
		Runs:         storage.NewWorkflowRunRepo(pool),
		ChatThreads:  storage.NewChatThreadRepo(pool),
		ChatMessages: storage.NewChatMessageRepo(pool),
		Engine:       executor.NewEngine(reg),
		Agent:        ag,
		Throttle:     auth.NewLoginThrottle(5, 15*time.Minute, time.Now),
		CookieDomain: "localhost",
		CookieSecure: false,
	})
	return srv, pool, "core.set"
}

type chatNopSetExec struct{}

func (c *chatNopSetExec) Execute(_ context.Context, in map[string]any) (registry.ExecuteResult, error) {
	v, _ := in["values"].(map[string]any)
	if v == nil {
		v = map[string]any{}
	}
	return registry.ExecuteResult{Output: v, Port: "main"}, nil
}

func signupAndCreateWF(t *testing.T, srv http.Handler) ([]*http.Cookie, string) {
	body, _ := json.Marshal(map[string]string{"email": "chat@example.com", "password": "supersecret"})
	r := httptest.NewRequest(http.MethodPost, "/v1/auth/signup", bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, r)
	cookies := w.Result().Cookies()

	wfBody, _ := json.Marshal(map[string]any{
		"name": "demo",
		"definition": map[string]any{
			"nodes": []any{
				map[string]any{"id": "a", "tool": "core.set", "params": map[string]any{"values": map[string]any{}}},
			},
			"edges": []any{},
		},
	})
	r2 := httptest.NewRequest(http.MethodPost, "/v1/workflows", bytes.NewReader(wfBody))
	r2.Header.Set("Content-Type", "application/json")
	for _, c := range cookies {
		r2.AddCookie(c)
	}
	w2 := httptest.NewRecorder()
	srv.ServeHTTP(w2, r2)
	var created struct{ ID string `json:"id"` }
	_ = json.Unmarshal(w2.Body.Bytes(), &created)
	return cookies, created.ID
}

func TestChat_proposesWorkflowAndStreamsSSE(t *testing.T) {
	mock := provider.NewMock()
	mock.Reply(provider.ChatResponse{
		Content: "Sure, here's the plan.",
		ToolCalls: []provider.ToolCall{{
			ID: "call_1", Name: "propose_workflow",
			Arguments: []byte(`{
				"name": "auto",
				"nodes": [{"id":"x","tool":"core.set","params":{"values":{}}}],
				"edges": []
			}`),
		}},
		StopReason: "tool_use",
	})

	srv, _, _ := newChatAPI(t, mock)
	cookies, wfID := signupAndCreateWF(t, srv)

	chatBody, _ := json.Marshal(map[string]any{
		"message": "Build me a workflow",
		"model":   "gpt-4o",
	})
	r := httptest.NewRequest(http.MethodPost, "/v1/workflows/"+wfID+"/chat", bytes.NewReader(chatBody))
	r.Header.Set("Content-Type", "application/json")
	for _, c := range cookies {
		r.AddCookie(c)
	}
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status: %d body: %s", w.Code, w.Body.String())
	}
	if ct := w.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/event-stream") {
		t.Errorf("content-type: %s", ct)
	}
	body := w.Body.String()
	if !strings.Contains(body, `"type":"proposal"`) {
		t.Errorf("missing proposal event: %s", body)
	}
	if !strings.Contains(body, "core.set") {
		t.Errorf("missing tool slug in proposal: %s", body)
	}
}
