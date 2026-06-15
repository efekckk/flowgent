package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/efekckk/flowgent/internal/auth"
	"github.com/efekckk/flowgent/internal/executor"
	"github.com/efekckk/flowgent/internal/idgen"
	"github.com/efekckk/flowgent/internal/storage"
)

const (
	sessionCookieName = "flowgent_session"
	sessionTTL        = 30 * 24 * time.Hour
	minPasswordLen    = 8
	maxPasswordLen    = 200
)

type Deps struct {
	Users        *storage.UserRepo
	Workspaces   *storage.WorkspaceRepo
	Sessions     *storage.SessionRepo
	Workflows    *storage.WorkflowRepo
	Runs         *storage.WorkflowRunRepo
	Engine       *executor.Engine
	Throttle     *auth.LoginThrottle
	CookieDomain string
	CookieSecure bool
}

type signupRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type signupResponse struct {
	User      userDTO      `json:"user"`
	Workspace workspaceDTO `json:"workspace"`
}

type userDTO struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}

type workspaceDTO struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func (d *Deps) handleSignup(w http.ResponseWriter, r *http.Request) {
	var req signupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid_json", "Request body must be JSON.")
		return
	}
	email := strings.ToLower(strings.TrimSpace(req.Email))
	if !looksLikeEmail(email) {
		WriteError(w, http.StatusBadRequest, "invalid_email", "Email is not a valid address.")
		return
	}
	if n := len(req.Password); n < minPasswordLen || n > maxPasswordLen {
		WriteError(w, http.StatusBadRequest, "invalid_password",
			"Password must be between 8 and 200 characters.")
		return
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "hash_failed", "Internal error.")
		return
	}

	ctx := r.Context()
	u := storage.User{ID: idgen.NewUser(), Email: email, PasswordHash: hash}
	if err := d.Users.Insert(ctx, u); err != nil {
		if errors.Is(err, storage.ErrConflict) {
			WriteError(w, http.StatusConflict, "email_taken", "Email already in use.")
			return
		}
		WriteError(w, http.StatusInternalServerError, "user_insert_failed", "Could not create user.")
		return
	}
	ws := storage.Workspace{ID: idgen.NewWorkspace(), OwnerUserID: u.ID, Name: "Default workspace"}
	if err := d.Workspaces.Insert(ctx, ws); err != nil {
		WriteError(w, http.StatusInternalServerError, "workspace_insert_failed", "Could not create workspace.")
		return
	}

	tok, err := d.issueSession(ctx, u.ID, r)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "session_issue_failed", "Could not start session.")
		return
	}
	d.setSessionCookie(w, tok)

	WriteJSON(w, http.StatusCreated, signupResponse{
		User:      userDTO{ID: u.ID, Email: u.Email},
		Workspace: workspaceDTO{ID: ws.ID, Name: ws.Name},
	})
}

func (d *Deps) issueSession(ctx context.Context, userID string, r *http.Request) (string, error) {
	tok, err := auth.GenerateSessionToken()
	if err != nil {
		return "", err
	}
	s := storage.Session{
		TokenHash: auth.HashSessionToken(tok),
		UserID:    userID,
		ExpiresAt: time.Now().Add(sessionTTL),
		IP:        clientIP(r),
		UserAgent: r.UserAgent(),
	}
	if err := d.Sessions.Insert(ctx, s); err != nil {
		return "", err
	}
	return tok, nil
}

func (d *Deps) setSessionCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		Domain:   d.CookieDomain,
		Expires:  time.Now().Add(sessionTTL),
		HttpOnly: true,
		Secure:   d.CookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
}

func looksLikeEmail(s string) bool {
	at := strings.IndexByte(s, '@')
	if at < 1 || at == len(s)-1 {
		return false
	}
	dot := strings.IndexByte(s[at+1:], '.')
	return dot > 0 && dot < len(s[at+1:])-1
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginResponse struct {
	User userDTO `json:"user"`
}

func (d *Deps) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid_json", "Request body must be JSON.")
		return
	}
	email := strings.ToLower(strings.TrimSpace(req.Email))

	ctx := r.Context()
	u, err := d.Users.FindByEmail(ctx, email)
	if err != nil {
		if !d.Throttle.AllowAndRecordFail(email) {
			WriteError(w, http.StatusTooManyRequests, "too_many_attempts",
				"Too many failed attempts. Try again in a few minutes.")
			return
		}
		WriteError(w, http.StatusUnauthorized, "invalid_credentials", "Email or password is wrong.")
		return
	}
	ok, err := auth.VerifyPassword(u.PasswordHash, req.Password)
	if err != nil || !ok {
		if !d.Throttle.AllowAndRecordFail(email) {
			WriteError(w, http.StatusTooManyRequests, "too_many_attempts",
				"Too many failed attempts. Try again in a few minutes.")
			return
		}
		WriteError(w, http.StatusUnauthorized, "invalid_credentials", "Email or password is wrong.")
		return
	}
	d.Throttle.Reset(email)

	tok, err := d.issueSession(ctx, u.ID, r)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "session_issue_failed", "Could not start session.")
		return
	}
	d.setSessionCookie(w, tok)
	WriteJSON(w, http.StatusOK, loginResponse{User: userDTO{ID: u.ID, Email: u.Email}})
}

func clientIP(r *http.Request) string {
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		if i := strings.IndexByte(fwd, ','); i > 0 {
			return strings.TrimSpace(fwd[:i])
		}
		return strings.TrimSpace(fwd)
	}
	host := r.RemoteAddr
	if i := strings.LastIndexByte(host, ':'); i > 0 {
		host = host[:i]
	}
	return host
}

func (d *Deps) handleMe(w http.ResponseWriter, r *http.Request) {
	u, ok := userFromContext(r.Context())
	if !ok {
		WriteError(w, http.StatusUnauthorized, "no_session", "Authentication required.")
		return
	}
	WriteJSON(w, http.StatusOK, map[string]any{
		"user": userDTO{ID: u.ID, Email: u.Email},
	})
}

func (d *Deps) handleLogout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(sessionCookieName)
	if err == nil {
		_ = d.Sessions.Delete(r.Context(), auth.HashSessionToken(cookie.Value))
	}
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		Domain:   d.CookieDomain,
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   d.CookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
	w.WriteHeader(http.StatusNoContent)
}
