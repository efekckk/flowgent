package api

import (
	"context"
	"net/http"

	"github.com/efekckk/flowgent/internal/auth"
	"github.com/efekckk/flowgent/internal/storage"
)

type ctxKey int

const ctxUserKey ctxKey = iota

// userFromContext returns the authenticated user, or false if none. Use
// SessionMiddleware to populate it on protected routes.
func userFromContext(ctx context.Context) (storage.User, bool) {
	u, ok := ctx.Value(ctxUserKey).(storage.User)
	return u, ok
}

func (d *Deps) SessionMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(sessionCookieName)
		if err != nil {
			WriteError(w, http.StatusUnauthorized, "no_session", "Authentication required.")
			return
		}
		hash := auth.HashSessionToken(cookie.Value)
		sess, err := d.Sessions.FindByTokenHash(r.Context(), hash)
		if err != nil {
			WriteError(w, http.StatusUnauthorized, "invalid_session", "Authentication required.")
			return
		}
		u, err := d.Users.FindByID(r.Context(), sess.UserID)
		if err != nil {
			WriteError(w, http.StatusUnauthorized, "invalid_session", "Authentication required.")
			return
		}
		_ = d.Sessions.Touch(r.Context(), hash)
		ctx := context.WithValue(r.Context(), ctxUserKey, u)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
