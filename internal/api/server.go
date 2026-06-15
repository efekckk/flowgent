package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func NewServer(d Deps) http.Handler {
	r := chi.NewRouter()
	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	r.Route("/v1/auth", func(sub chi.Router) {
		sub.Post("/signup", d.handleSignup)
		sub.Post("/login", d.handleLogin)
		sub.Post("/logout", d.handleLogout)
	})
	r.Route("/v1", func(sub chi.Router) {
		sub.Use(d.SessionMiddleware)
		sub.Get("/me", d.handleMe)
		sub.Post("/workflows", d.handleCreateWorkflow)
		sub.Get("/workflows/{id}", d.handleGetWorkflow)
		sub.Post("/workflows/{id}/run", d.handleRunWorkflow)
	})
	return r
}
