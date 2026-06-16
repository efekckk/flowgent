package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/efekckk/flowgent/internal/webfs"
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
		sub.Post("/workflows/{id}/chat", d.handleChat)
		sub.Post("/credentials", d.handleCreateCredential)
		sub.Get("/credentials", d.handleListCredentials)
		sub.Delete("/credentials/{id}", d.handleDeleteCredential)
		sub.Post("/workflows/{id}/triggers", d.handleCreateTrigger)
		sub.Get("/workflows/{id}/triggers", d.handleListTriggers)
		sub.Patch("/triggers/{id}", d.handleUpdateTrigger)
		sub.Delete("/triggers/{id}", d.handleDeleteTrigger)
		sub.Get("/workflows/{id}/runs", d.handleListRunsForWorkflow)
		sub.Get("/runs/{id}", d.handleGetRun)
		sub.Get("/runs/{id}/logs", d.handleGetRunLogs)
		sub.Get("/runs/{id}/stream", d.handleStreamRun)
		sub.Post("/runs/{id}/replay", d.handleReplayRun)
		sub.Get("/workspaces/{wsID}/runs/search", d.handleSearchRunLogs)
	})
	// Webhook ingress: unauthenticated by design — external services hit
	// /webhooks/{trigger_id}/{token} directly and authenticate via the
	// URL token plus an optional HMAC signature on the body. Mounted
	// outside the /v1 group so the session middleware does not run.
	if d.WebhookHandler != nil {
		r.Post("/webhooks/{trigger_id}/{token}", d.WebhookHandler.ServeHTTP)
	}
	// Static SPA at root — must be last so existing /v1 and /health win.
	r.Mount("/", webfs.Handler())
	return r
}
