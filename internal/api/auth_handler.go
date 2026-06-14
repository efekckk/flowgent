package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/efekckk/flowgent/internal/auth"
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
