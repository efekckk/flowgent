package api

import (
	"context"
	"errors"
	"net/http"

	"github.com/efekckk/flowgent/internal/storage"
)

// Ownership guards keep one workspace's data out of another's reach. Every
// /v1 handler that takes a {workflow_id}, {trigger_id}, {run_id}, or
// {credential_id} path param must resolve the resource through one of these
// before touching it: load the resource, walk to the workspace, confirm the
// session user owns that workspace. On any mismatch the handler returns 404
// "not found" rather than 403 so the existence of resources in other
// tenants stays hidden.

// userOwnsWorkspace reports whether the user owns the given workspace id.
// A nil/empty workspaceID is treated as not owned.
func (d *Deps) userOwnsWorkspace(ctx context.Context, userID, workspaceID string) (bool, error) {
	if workspaceID == "" {
		return false, nil
	}
	rows, err := d.Workspaces.FindByOwner(ctx, userID)
	if err != nil {
		return false, err
	}
	for _, ws := range rows {
		if ws.ID == workspaceID {
			return true, nil
		}
	}
	return false, nil
}

// loadOwnedWorkflow fetches the workflow at id and writes a 404 if either
// the row is missing or the session user doesn't own its workspace. The
// caller uses the returned ok bool to short-circuit.
func (d *Deps) loadOwnedWorkflow(w http.ResponseWriter, r *http.Request, workflowID string) (storage.Workflow, bool) {
	u, ok := userFromContext(r.Context())
	if !ok {
		WriteError(w, http.StatusUnauthorized, "no_session", "Authentication required.")
		return storage.Workflow{}, false
	}
	wf, err := d.Workflows.Get(r.Context(), workflowID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			WriteError(w, http.StatusNotFound, "not_found", "Workflow not found.")
			return storage.Workflow{}, false
		}
		WriteError(w, http.StatusInternalServerError, "get_failed", "Could not fetch workflow.")
		return storage.Workflow{}, false
	}
	owned, err := d.userOwnsWorkspace(r.Context(), u.ID, wf.WorkspaceID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "workspace_lookup_failed", "Could not load workspace.")
		return storage.Workflow{}, false
	}
	if !owned {
		// Mask cross-tenant existence as plain 404.
		WriteError(w, http.StatusNotFound, "not_found", "Workflow not found.")
		return storage.Workflow{}, false
	}
	return wf, true
}

// loadOwnedTrigger fetches the trigger at id and verifies the session user
// owns the workflow's workspace. Used by PATCH/DELETE /v1/triggers/{id}.
func (d *Deps) loadOwnedTrigger(w http.ResponseWriter, r *http.Request, triggerID string) (storage.Trigger, bool) {
	u, ok := userFromContext(r.Context())
	if !ok {
		WriteError(w, http.StatusUnauthorized, "no_session", "Authentication required.")
		return storage.Trigger{}, false
	}
	trg, err := d.Triggers.Get(r.Context(), triggerID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			WriteError(w, http.StatusNotFound, "not_found", "Trigger not found.")
			return storage.Trigger{}, false
		}
		WriteError(w, http.StatusInternalServerError, "get_failed", "Could not fetch trigger.")
		return storage.Trigger{}, false
	}
	wf, err := d.Workflows.Get(r.Context(), trg.WorkflowID)
	if err != nil {
		WriteError(w, http.StatusNotFound, "not_found", "Trigger not found.")
		return storage.Trigger{}, false
	}
	owned, err := d.userOwnsWorkspace(r.Context(), u.ID, wf.WorkspaceID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "workspace_lookup_failed", "Could not load workspace.")
		return storage.Trigger{}, false
	}
	if !owned {
		WriteError(w, http.StatusNotFound, "not_found", "Trigger not found.")
		return storage.Trigger{}, false
	}
	return trg, true
}

// loadOwnedRun fetches the run at id and verifies the session user owns
// the parent workflow's workspace. Used by every /v1/runs/{id}* handler.
func (d *Deps) loadOwnedRun(w http.ResponseWriter, r *http.Request, runID string) (storage.WorkflowRun, bool) {
	u, ok := userFromContext(r.Context())
	if !ok {
		WriteError(w, http.StatusUnauthorized, "no_session", "Authentication required.")
		return storage.WorkflowRun{}, false
	}
	run, err := d.Runs.Get(r.Context(), runID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			WriteError(w, http.StatusNotFound, "not_found", "Run not found.")
			return storage.WorkflowRun{}, false
		}
		WriteError(w, http.StatusInternalServerError, "get_failed", "Could not fetch run.")
		return storage.WorkflowRun{}, false
	}
	wf, err := d.Workflows.Get(r.Context(), run.WorkflowID)
	if err != nil {
		WriteError(w, http.StatusNotFound, "not_found", "Run not found.")
		return storage.WorkflowRun{}, false
	}
	owned, err := d.userOwnsWorkspace(r.Context(), u.ID, wf.WorkspaceID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "workspace_lookup_failed", "Could not load workspace.")
		return storage.WorkflowRun{}, false
	}
	if !owned {
		WriteError(w, http.StatusNotFound, "not_found", "Run not found.")
		return storage.WorkflowRun{}, false
	}
	return run, true
}
