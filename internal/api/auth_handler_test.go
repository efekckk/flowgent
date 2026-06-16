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

func TestMain(m *testing.M) {
	// Ensure storagetest container starts when Docker is available; otherwise
	// individual tests will skip via the Fresh() call. Mirror the pattern used
	// in internal/storage/main_test.go.
	if err := storagetest.Start(); err == nil {
		defer storagetest.Stop()
	}
	m.Run()
}

func newAPI(t *testing.T) (http.Handler, *pgxpool.Pool) {
	t.Helper()
	dsn := storagetest.Fresh(t)
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	t.Cleanup(pool.Close)
	srv := api.NewServer(api.Deps{
		Users:        storage.NewUserRepo(pool),
		Workspaces:   storage.NewWorkspaceRepo(pool),
		Sessions:     storage.NewSessionRepo(pool),
		Throttle:     auth.NewLoginThrottle(5, 15*time.Minute, time.Now),
		CookieDomain: "localhost",
		CookieSecure: false,
	})
	return srv, pool
}

func TestSignup_createsUserWorkspaceAndSession(t *testing.T) {
	srv, pool := newAPI(t)

	body, _ := json.Marshal(map[string]string{
		"email":    "newuser@example.com",
		"password": "supersecret",
	})
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/signup", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("status: %d body=%s", rr.Code, rr.Body.String())
	}

	var resp struct {
		User struct {
			ID    string `json:"id"`
			Email string `json:"email"`
		} `json:"user"`
		Workspace struct {
			ID string `json:"id"`
		} `json:"workspace"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.User.Email != "newuser@example.com" {
		t.Errorf("email: %s", resp.User.Email)
	}

	cookies := rr.Result().Cookies()
	var sessionCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "flowgent_session" {
			sessionCookie = c
		}
	}
	if sessionCookie == nil {
		t.Fatalf("expected flowgent_session cookie, got: %+v", cookies)
	}
	if !sessionCookie.HttpOnly {
		t.Errorf("session cookie must be HttpOnly")
	}

	var cnt int
	_ = pool.QueryRow(context.Background(), `SELECT count(*) FROM users WHERE email=$1`, "newuser@example.com").Scan(&cnt)
	if cnt != 1 {
		t.Errorf("user row count: %d", cnt)
	}
}

func TestSignup_rejectsDuplicateEmail(t *testing.T) {
	srv, _ := newAPI(t)
	body, _ := json.Marshal(map[string]string{"email": "dup@example.com", "password": "x12345678"})

	r1 := httptest.NewRequest(http.MethodPost, "/v1/auth/signup", bytes.NewReader(body))
	r1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	srv.ServeHTTP(w1, r1)
	if w1.Code != http.StatusCreated {
		t.Fatalf("first signup: %d %s", w1.Code, w1.Body.String())
	}

	r2 := httptest.NewRequest(http.MethodPost, "/v1/auth/signup", bytes.NewReader(body))
	r2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	srv.ServeHTTP(w2, r2)
	if w2.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d body=%s", w2.Code, w2.Body.String())
	}
}

func TestSignup_rejectsShortPassword(t *testing.T) {
	srv, _ := newAPI(t)
	body, _ := json.Marshal(map[string]string{"email": "short@example.com", "password": "abc"})
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/signup", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestLogin_acceptsCorrectCredentials(t *testing.T) {
	srv, _ := newAPI(t)
	body, _ := json.Marshal(map[string]string{"email": "login@example.com", "password": "supersecret"})
	r1 := httptest.NewRequest(http.MethodPost, "/v1/auth/signup", bytes.NewReader(body))
	r1.Header.Set("Content-Type", "application/json")
	srv.ServeHTTP(httptest.NewRecorder(), r1)

	r2 := httptest.NewRequest(http.MethodPost, "/v1/auth/login", bytes.NewReader(body))
	r2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	srv.ServeHTTP(w2, r2)
	if w2.Code != http.StatusOK {
		t.Fatalf("status: %d body=%s", w2.Code, w2.Body.String())
	}
	if len(w2.Result().Cookies()) == 0 {
		t.Fatalf("expected session cookie")
	}
}

func TestLogin_rejectsWrongPassword(t *testing.T) {
	srv, _ := newAPI(t)
	signupBody, _ := json.Marshal(map[string]string{"email": "wrong@example.com", "password": "supersecret"})
	r1 := httptest.NewRequest(http.MethodPost, "/v1/auth/signup", bytes.NewReader(signupBody))
	r1.Header.Set("Content-Type", "application/json")
	srv.ServeHTTP(httptest.NewRecorder(), r1)

	loginBody, _ := json.Marshal(map[string]string{"email": "wrong@example.com", "password": "notright"})
	r2 := httptest.NewRequest(http.MethodPost, "/v1/auth/login", bytes.NewReader(loginBody))
	r2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	srv.ServeHTTP(w2, r2)
	if w2.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body=%s", w2.Code, w2.Body.String())
	}
}

func TestLogin_unknownEmailIsAlso401(t *testing.T) {
	srv, _ := newAPI(t)
	body, _ := json.Marshal(map[string]string{"email": "nobody@example.com", "password": "supersecret"})
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status: %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestMe_returnsCurrentUserWhenSessionValid(t *testing.T) {
	srv, _ := newAPI(t)
	body, _ := json.Marshal(map[string]string{"email": "me@example.com", "password": "supersecret"})
	r1 := httptest.NewRequest(http.MethodPost, "/v1/auth/signup", bytes.NewReader(body))
	r1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	srv.ServeHTTP(w1, r1)
	cookies := w1.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatalf("no session cookie")
	}

	r2 := httptest.NewRequest(http.MethodGet, "/v1/me", nil)
	for _, c := range cookies {
		r2.AddCookie(c)
	}
	w2 := httptest.NewRecorder()
	srv.ServeHTTP(w2, r2)
	if w2.Code != http.StatusOK {
		t.Fatalf("status: %d body=%s", w2.Code, w2.Body.String())
	}
	if !bytes.Contains(w2.Body.Bytes(), []byte("me@example.com")) {
		t.Errorf("body missing email: %s", w2.Body.String())
	}
	// /v1/me must surface the user's primary workspace so the SPA can call
	// workspace-scoped endpoints (e.g. run log search) on first paint.
	if !bytes.Contains(w2.Body.Bytes(), []byte(`"workspace"`)) {
		t.Errorf("body missing workspace: %s", w2.Body.String())
	}
}

func TestMe_returns401WithoutSession(t *testing.T) {
	srv, _ := newAPI(t)
	req := httptest.NewRequest(http.MethodGet, "/v1/me", nil)
	rr := httptest.NewRecorder()
	srv.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status: %d", rr.Code)
	}
}

func TestLogin_lockoutAfter5Failures(t *testing.T) {
	srv, _ := newAPI(t)
	body, _ := json.Marshal(map[string]string{"email": "lock@example.com", "password": "supersecret"})
	r1 := httptest.NewRequest(http.MethodPost, "/v1/auth/signup", bytes.NewReader(body))
	r1.Header.Set("Content-Type", "application/json")
	srv.ServeHTTP(httptest.NewRecorder(), r1)

	wrong, _ := json.Marshal(map[string]string{"email": "lock@example.com", "password": "wrong-one"})
	for i := 0; i < 5; i++ {
		r := httptest.NewRequest(http.MethodPost, "/v1/auth/login", bytes.NewReader(wrong))
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		srv.ServeHTTP(w, r)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("attempt %d: expected 401, got %d", i+1, w.Code)
		}
	}
	r6 := httptest.NewRequest(http.MethodPost, "/v1/auth/login", bytes.NewReader(wrong))
	r6.Header.Set("Content-Type", "application/json")
	w6 := httptest.NewRecorder()
	srv.ServeHTTP(w6, r6)
	if w6.Code != http.StatusTooManyRequests {
		t.Fatalf("6th attempt: expected 429, got %d body=%s", w6.Code, w6.Body.String())
	}
}

func TestLogout_invalidatesSession(t *testing.T) {
	srv, _ := newAPI(t)
	body, _ := json.Marshal(map[string]string{"email": "out@example.com", "password": "supersecret"})
	r1 := httptest.NewRequest(http.MethodPost, "/v1/auth/signup", bytes.NewReader(body))
	r1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	srv.ServeHTTP(w1, r1)
	cookies := w1.Result().Cookies()

	r2 := httptest.NewRequest(http.MethodPost, "/v1/auth/logout", nil)
	for _, c := range cookies {
		r2.AddCookie(c)
	}
	w2 := httptest.NewRecorder()
	srv.ServeHTTP(w2, r2)
	if w2.Code != http.StatusNoContent {
		t.Fatalf("logout status: %d", w2.Code)
	}

	r3 := httptest.NewRequest(http.MethodGet, "/v1/me", nil)
	for _, c := range cookies {
		r3.AddCookie(c)
	}
	w3 := httptest.NewRecorder()
	srv.ServeHTTP(w3, r3)
	if w3.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 after logout, got %d", w3.Code)
	}
}
