package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/efekckk/flowgent/internal/executor"
	"github.com/efekckk/flowgent/internal/idgen"
	"github.com/efekckk/flowgent/internal/storage"
)

type createWorkflowRequest struct {
	Name       string          `json:"name"`
	Definition json.RawMessage `json:"definition"`
}

type workflowDTO struct {
	ID         string          `json:"id"`
	Name       string          `json:"name"`
	Status     string          `json:"status"`
	Version    int             `json:"version"`
	Definition json.RawMessage `json:"definition,omitempty"`
}

func (d *Deps) handleCreateWorkflow(w http.ResponseWriter, r *http.Request) {
	u, ok := userFromContext(r.Context())
	if !ok {
		WriteError(w, http.StatusUnauthorized, "no_session", "Authentication required.")
		return
	}
	var req createWorkflowRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid_json", "Request body must be JSON.")
		return
	}
	if req.Name == "" {
		WriteError(w, http.StatusBadRequest, "invalid_name", "Workflow name is required.")
		return
	}
	if len(req.Definition) == 0 {
		WriteError(w, http.StatusBadRequest, "invalid_definition", "Workflow definition is required.")
		return
	}

	workspaces, err := d.Workspaces.FindByOwner(r.Context(), u.ID)
	if err != nil || len(workspaces) == 0 {
		WriteError(w, http.StatusInternalServerError, "workspace_missing", "No workspace found.")
		return
	}
	wf := storage.Workflow{
		ID:             idgen.NewWorkflow(),
		WorkspaceID:    workspaces[0].ID,
		Name:           req.Name,
		Status:         "draft",
		CurrentVersion: 1,
	}
	if err := d.Workflows.Insert(r.Context(), wf); err != nil {
		WriteError(w, http.StatusInternalServerError, "workflow_insert_failed", "Could not create workflow.")
		return
	}
	if err := d.Workflows.SaveVersion(r.Context(), storage.WorkflowVersion{
		ID:              idgen.NewWorkflowVersion(),
		WorkflowID:      wf.ID,
		Version:         1,
		Definition:      req.Definition,
		Message:         "initial",
		CreatedByUserID: &u.ID,
	}); err != nil {
		WriteError(w, http.StatusInternalServerError, "version_save_failed", "Could not save version.")
		return
	}
	WriteJSON(w, http.StatusCreated, workflowDTO{
		ID: wf.ID, Name: wf.Name, Status: wf.Status, Version: 1,
	})
}

func (d *Deps) handleGetWorkflow(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	wf, ok := d.loadOwnedWorkflow(w, r, id)
	if !ok {
		return
	}
	ver, err := d.Workflows.GetVersion(r.Context(), wf.ID, wf.CurrentVersion)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "version_get_failed", "Could not fetch current version.")
		return
	}
	WriteJSON(w, http.StatusOK, workflowDTO{
		ID: wf.ID, Name: wf.Name, Status: wf.Status, Version: wf.CurrentVersion, Definition: ver.Definition,
	})
}

type runResponse struct {
	RunID  string `json:"run_id"`
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

func (d *Deps) handleRunWorkflow(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	wf, ok := d.loadOwnedWorkflow(w, r, id)
	if !ok {
		return
	}
	ver, err := d.Workflows.GetVersion(r.Context(), wf.ID, wf.CurrentVersion)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "version_get_failed", "Could not fetch version.")
		return
	}

	var def executor.Workflow
	if err := json.Unmarshal(ver.Definition, &def); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid_definition", "Stored definition is not valid JSON.")
		return
	}

	var triggerPayload map[string]any
	if r.ContentLength > 0 {
		_ = json.NewDecoder(r.Body).Decode(&triggerPayload)
	}

	runID := idgen.NewRun()
	now := time.Now().UTC()
	payloadBytes, _ := json.Marshal(triggerPayload)
	if err := d.Runs.NewRun(r.Context(), storage.WorkflowRun{
		ID:              runID,
		WorkflowID:      wf.ID,
		WorkflowVersion: wf.CurrentVersion,
		Status:          "running",
		TriggerKind:     "manual",
		TriggerPayload:  payloadBytes,
		StartedAt:       &now,
	}); err != nil {
		WriteError(w, http.StatusInternalServerError, "run_create_failed", "Could not start run.")
		return
	}

	// Defensive: if we exit early (panic, client disconnect before
	// UpdateRunStatus, etc.) the row would stay in 'running'. The CAS
	// fail-if-running guard makes this safe to invoke unconditionally.
	settled := false
	defer func() {
		if settled {
			return
		}
		bgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = d.Runs.FailRunIfRunning(bgCtx, runID, "handler exited before final status", time.Now().UTC())
	}()

	state, runErr := d.Engine.Run(r.Context(), &def, executor.RunOptions{
		TriggerKind:    "manual",
		TriggerPayload: triggerPayload,
	})
	finishedAt := time.Now().UTC()
	persistNodeRuns(r.Context(), d.Runs, runID, &def, state)
	runStatus, runErrMsg := state.RunStatus()
	errText := runErrMsg
	if runErr != nil && errText == "" {
		errText = runErr.Error()
	}
	_ = d.Runs.UpdateRunStatus(r.Context(), runID, runStatus, errText, &now, &finishedAt)
	settled = true

	resp := runResponse{RunID: runID, Status: runStatus, Error: errText}
	WriteJSON(w, http.StatusOK, resp)
}

func persistNodeRuns(ctx context.Context, repo *storage.WorkflowRunRepo, runID string, wf *executor.Workflow, state *executor.RunState) {
	for _, node := range wf.Nodes {
		records := state.History(node.ID)
		if len(records) == 0 {
			status := state.Status(node.ID)
			if status == "" {
				status = "skipped"
			}
			_ = repo.InsertNodeRun(ctx, storage.NodeRun{
				ID:            idgen.NewNodeRun(),
				WorkflowRunID: runID,
				NodeID:        node.ID,
				Iteration:     0,
				Status:        status,
				Attempts:      0,
			})
			continue
		}
		inputBytes, _ := json.Marshal(state.Input(node.ID))
		for _, rec := range records {
			outputBytes, _ := json.Marshal(rec.Output)
			_ = repo.InsertNodeRun(ctx, storage.NodeRun{
				ID:            idgen.NewNodeRun(),
				WorkflowRunID: runID,
				NodeID:        node.ID,
				Iteration:     rec.Iteration,
				Status:        rec.Status,
				Input:         inputBytes,
				Output:        outputBytes,
				Attempts:      rec.Attempts,
			})
		}
	}
}
