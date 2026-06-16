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
	"github.com/efekckk/flowgent/internal/storage"
	"github.com/efekckk/flowgent/internal/storage/storagetest"
)

func newCredsAPI(t *testing.T) (http.Handler, *pgxpool.Pool) {
	t.Helper()
	dsn := storagetest.Fresh(t)
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	t.Cleanup(pool.Close)

	masterKey := make([]byte, 32)
	for i := range masterKey {
		masterKey[i] = byte(i)
	}

	srv := api.NewServer(api.Deps{
		Users:         storage.NewUserRepo(pool),
		Workspaces:    storage.NewWorkspaceRepo(pool),
		Sessions:      storage.NewSessionRepo(pool),
		Credentials:   storage.NewCredentialRepo(pool),
		Throttle:      auth.NewLoginThrottle(5, 15*time.Minute, time.Now),
		CredentialKey: masterKey,
		CookieDomain:  "localhost",
		CookieSecure:  false,
	})
	return srv, pool
}

func signupForCreds(t *testing.T, srv http.Handler) []*http.Cookie {
	body, _ := json.Marshal(map[string]string{"email": "cr@example.com", "password": "supersecret"})
	r := httptest.NewRequest(http.MethodPost, "/v1/auth/signup", bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, r)
	if w.Code != http.StatusCreated {
		t.Fatalf("signup: %d %s", w.Code, w.Body.String())
	}
	return w.Result().Cookies()
}

func TestCreateCredential_encryptsAndReturnsId(t *testing.T) {
	srv, pool := newCredsAPI(t)
	cookies := signupForCreds(t, srv)

	body, _ := json.Marshal(map[string]any{
		"name":   "openai_default",
		"type":   "openai",
		"secret": map[string]any{"api_key": "sk-test-1234"},
	})
	r := httptest.NewRequest(http.MethodPost, "/v1/credentials", bytes.NewReader(body))
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
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.ID == "" || resp.Name != "openai_default" {
		t.Errorf("resp: %+v", resp)
	}

	var encrypted []byte
	_ = pool.QueryRow(context.Background(),
		`SELECT encrypted FROM credentials WHERE id=$1`, resp.ID).Scan(&encrypted)
	if bytes.Contains(encrypted, []byte("sk-test-1234")) {
		t.Errorf("plaintext leaked into encrypted column")
	}
}

func TestListCredentials_returnsRowsForWorkspace(t *testing.T) {
	srv, _ := newCredsAPI(t)
	cookies := signupForCreds(t, srv)

	for _, name := range []string{"a", "b", "c"} {
		body, _ := json.Marshal(map[string]any{
			"name":   name, "type": "openai",
			"secret": map[string]any{"api_key": "sk"},
		})
		r := httptest.NewRequest(http.MethodPost, "/v1/credentials", bytes.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		for _, c := range cookies {
			r.AddCookie(c)
		}
		srv.ServeHTTP(httptest.NewRecorder(), r)
	}

	r := httptest.NewRequest(http.MethodGet, "/v1/credentials", nil)
	for _, c := range cookies {
		r.AddCookie(c)
	}
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("status: %d", w.Code)
	}
	var resp struct {
		Items []map[string]any `json:"items"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if len(resp.Items) != 3 {
		t.Errorf("items: %d", len(resp.Items))
	}
}

func TestDeleteCredential_removesRow(t *testing.T) {
	srv, _ := newCredsAPI(t)
	cookies := signupForCreds(t, srv)

	body, _ := json.Marshal(map[string]any{
		"name": "kill", "type": "openai", "secret": map[string]any{"api_key": "sk"},
	})
	r1 := httptest.NewRequest(http.MethodPost, "/v1/credentials", bytes.NewReader(body))
	r1.Header.Set("Content-Type", "application/json")
	for _, c := range cookies {
		r1.AddCookie(c)
	}
	w1 := httptest.NewRecorder()
	srv.ServeHTTP(w1, r1)
	var created struct{ ID string `json:"id"` }
	_ = json.Unmarshal(w1.Body.Bytes(), &created)

	r2 := httptest.NewRequest(http.MethodDelete, "/v1/credentials/"+created.ID, nil)
	for _, c := range cookies {
		r2.AddCookie(c)
	}
	w2 := httptest.NewRecorder()
	srv.ServeHTTP(w2, r2)
	if w2.Code != http.StatusNoContent {
		t.Fatalf("delete status: %d", w2.Code)
	}

	r3 := httptest.NewRequest(http.MethodGet, "/v1/credentials", nil)
	for _, c := range cookies {
		r3.AddCookie(c)
	}
	w3 := httptest.NewRecorder()
	srv.ServeHTTP(w3, r3)
	var resp struct {
		Items []map[string]any `json:"items"`
	}
	_ = json.Unmarshal(w3.Body.Bytes(), &resp)
	if len(resp.Items) != 0 {
		t.Errorf("expected 0 after delete, got %d", len(resp.Items))
	}
}

func TestCreateCredential_requiresAuth(t *testing.T) {
	srv, _ := newCredsAPI(t)
	body, _ := json.Marshal(map[string]any{"name": "x", "type": "openai", "secret": map[string]any{}})
	r := httptest.NewRequest(http.MethodPost, "/v1/credentials", bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status: %d", w.Code)
	}
}
