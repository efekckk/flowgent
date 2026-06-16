package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// handleSearchRunLogs exposes workspace-scoped full-text search over
// run_logs. The current user's workspace must match the path param to
// prevent cross-tenant data exposure. Query strings shorter than 3 chars
// are rejected to keep tsvector latency predictable on large workspaces.
func (d *Deps) handleSearchRunLogs(w http.ResponseWriter, r *http.Request) {
	u, ok := userFromContext(r.Context())
	if !ok {
		WriteError(w, http.StatusUnauthorized, "no_session", "Authentication required.")
		return
	}
	wsID := chi.URLParam(r, "wsID")
	if wsID == "" {
		WriteError(w, http.StatusBadRequest, "invalid_input", "Workspace id is required.")
		return
	}

	// Workspace scope check mirrors credential_handler.go: derive owned
	// workspaces for the session user and require a match. A mismatch is
	// 403 (not 404) per the spec — we don't want to leak whether the
	// workspace id exists in another tenant.
	workspaces, err := d.Workspaces.FindByOwner(r.Context(), u.ID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "workspace_lookup_failed", "Could not load workspace.")
		return
	}
	owned := false
	for _, ws := range workspaces {
		if ws.ID == wsID {
			owned = true
			break
		}
	}
	if !owned {
		WriteError(w, http.StatusForbidden, "forbidden", "Workspace is not accessible.")
		return
	}

	q := r.URL.Query().Get("q")
	if len(q) < 3 {
		WriteError(w, http.StatusBadRequest, "invalid_query", "Query must be at least 3 characters.")
		return
	}
	limit := parseIntDefault(r.URL.Query().Get("limit"), 50, 100)

	if d.RunLogs == nil {
		WriteError(w, http.StatusServiceUnavailable, "search_disabled", "Search is not enabled.")
		return
	}
	hits, err := d.RunLogs.Search(r.Context(), wsID, q, limit)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "search_failed", "Could not search run logs.")
		return
	}
	WriteJSON(w, http.StatusOK, map[string]any{"hits": hits})
}
