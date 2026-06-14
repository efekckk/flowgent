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
	})
	return r
}
