package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/efekckk/flowgent/internal/storage"
)

// runDTO is the on-the-wire shape of a workflow_run. We deliberately do
// not embed storage.WorkflowRun: callers shouldn't have to learn pgx
// types, and we want to keep the JSON field names stable as the
// underlying schema evolves.
type runDTO struct {
	ID              string          `json:"id"`
	WorkflowID      string          `json:"workflow_id"`
	WorkflowVersion int             `json:"workflow_version"`
	Status          string          `json:"status"`
	TriggerKind     string          `json:"trigger_kind"`
	TriggerID       *string         `json:"trigger_id,omitempty"`
	ParentRunID     *string         `json:"parent_run_id,omitempty"`
	TriggerPayload  json.RawMessage `json:"trigger_payload,omitempty"`
	Error           string          `json:"error,omitempty"`
	StartedAt       *time.Time      `json:"started_at,omitempty"`
	FinishedAt      *time.Time      `json:"finished_at,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
}

type nodeRunDTO struct {
	ID         string          `json:"id"`
	NodeID     string          `json:"node_id"`
	Iteration  int             `json:"iteration"`
	Status     string          `json:"status"`
	Input      json.RawMessage `json:"input,omitempty"`
	Output     json.RawMessage `json:"output,omitempty"`
	Error      json.RawMessage `json:"error,omitempty"`
	Attempts   int             `json:"attempts"`
	StartedAt  *time.Time      `json:"started_at,omitempty"`
	FinishedAt *time.Time      `json:"finished_at,omitempty"`
	DurationMs *int            `json:"duration_ms,omitempty"`
}

type runLogDTO struct {
	ID      int64     `json:"id"`
	RunID   string    `json:"run_id"`
	NodeID  string    `json:"node_id,omitempty"`
	Level   string    `json:"level"`
	Message string    `json:"message"`
	At      time.Time `json:"at"`
}

func toRunDTO(r storage.WorkflowRun) runDTO {
	d := runDTO{
		ID:              r.ID,
		WorkflowID:      r.WorkflowID,
		WorkflowVersion: r.WorkflowVersion,
		Status:          r.Status,
		TriggerKind:     r.TriggerKind,
		TriggerID:       r.TriggerID,
		ParentRunID:     r.ParentRunID,
		TriggerPayload:  r.TriggerPayload,
		Error:           r.Error,
		StartedAt:       r.StartedAt,
		FinishedAt:      r.FinishedAt,
		CreatedAt:       r.CreatedAt,
	}
	return d
}

func toNodeRunDTO(n storage.NodeRun) nodeRunDTO {
	return nodeRunDTO{
		ID:         n.ID,
		NodeID:     n.NodeID,
		Iteration:  n.Iteration,
		Status:     n.Status,
		Input:      n.Input,
		Output:     n.Output,
		Error:      n.Error,
		Attempts:   n.Attempts,
		StartedAt:  n.StartedAt,
		FinishedAt: n.FinishedAt,
		DurationMs: n.DurationMs,
	}
}

// handleListRunsForWorkflow paginates workflow_runs newest-first. status,
// from, and to query params are optional filters; cursor opaquely encodes
// the previous page's tail.
func (d *Deps) handleListRunsForWorkflow(w http.ResponseWriter, r *http.Request) {
	if _, ok := userFromContext(r.Context()); !ok {
		WriteError(w, http.StatusUnauthorized, "no_session", "Authentication required.")
		return
	}
	wfID := chi.URLParam(r, "id")
	if wfID == "" {
		WriteError(w, http.StatusBadRequest, "invalid_input", "Workflow id is required.")
		return
	}
	q := r.URL.Query()
	filter := storage.RunFilter{
		Status: q.Get("status"),
		Cursor: q.Get("cursor"),
		Limit:  parseIntDefault(q.Get("limit"), 50, 200),
	}
	if from := q.Get("from"); from != "" {
		if t, err := time.Parse(time.RFC3339, from); err == nil {
			filter.From = &t
		}
	}
	if to := q.Get("to"); to != "" {
		if t, err := time.Parse(time.RFC3339, to); err == nil {
			filter.To = &t
		}
	}

	items, nextCursor, err := d.Runs.ListForWorkflow(r.Context(), wfID, filter)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "list_failed", "Could not list runs.")
		return
	}
	out := make([]runDTO, 0, len(items))
	for _, it := range items {
		out = append(out, toRunDTO(it))
	}
	WriteJSON(w, http.StatusOK, map[string]any{
		"items":       out,
		"next_cursor": nextCursor,
	})
}

// handleGetRun returns one run plus all its node_runs in a single response;
// the run viewer wants both halves and a single round-trip simplifies the
// client.
func (d *Deps) handleGetRun(w http.ResponseWriter, r *http.Request) {
	if _, ok := userFromContext(r.Context()); !ok {
		WriteError(w, http.StatusUnauthorized, "no_session", "Authentication required.")
		return
	}
	id := chi.URLParam(r, "id")
	run, nodes, err := d.Runs.GetWithNodes(r.Context(), id)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			WriteError(w, http.StatusNotFound, "not_found", "Run not found.")
			return
		}
		WriteError(w, http.StatusInternalServerError, "get_failed", "Could not fetch run.")
		return
	}
	nodeDTOs := make([]nodeRunDTO, 0, len(nodes))
	for _, n := range nodes {
		nodeDTOs = append(nodeDTOs, toNodeRunDTO(n))
	}
	WriteJSON(w, http.StatusOK, map[string]any{
		"run":   toRunDTO(run),
		"nodes": nodeDTOs,
	})
}

// handleGetRunLogs serves a paginated tail of run_logs. The since query
// param holds the last id the caller observed, so SPA polling remains
// O(new rows) instead of O(total log size).
func (d *Deps) handleGetRunLogs(w http.ResponseWriter, r *http.Request) {
	if _, ok := userFromContext(r.Context()); !ok {
		WriteError(w, http.StatusUnauthorized, "no_session", "Authentication required.")
		return
	}
	if d.RunLogs == nil {
		WriteError(w, http.StatusServiceUnavailable, "logs_disabled", "Run logs are not enabled.")
		return
	}
	id := chi.URLParam(r, "id")
	since, _ := strconv.ParseInt(r.URL.Query().Get("since"), 10, 64)
	limit := parseIntDefault(r.URL.Query().Get("limit"), 200, 1000)
	items, err := d.RunLogs.ListByRun(r.Context(), id, since, limit)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "list_failed", "Could not list logs.")
		return
	}
	dtos := make([]runLogDTO, 0, len(items))
	for _, it := range items {
		dtos = append(dtos, runLogDTO{
			ID:      it.ID,
			RunID:   it.RunID,
			NodeID:  it.NodeID,
			Level:   it.Level,
			Message: it.Message,
			At:      it.At,
		})
	}
	WriteJSON(w, http.StatusOK, map[string]any{"items": dtos})
}

// handleReplayRun forks a brand-new run that inherits the parent's
// trigger_payload. The engine's RunFromReplay owns the workflow lookup +
// persistence — this handler is just the HTTP front door.
func (d *Deps) handleReplayRun(w http.ResponseWriter, r *http.Request) {
	if _, ok := userFromContext(r.Context()); !ok {
		WriteError(w, http.StatusUnauthorized, "no_session", "Authentication required.")
		return
	}
	id := chi.URLParam(r, "id")
	parent, _, err := d.Runs.GetWithNodes(r.Context(), id)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			WriteError(w, http.StatusNotFound, "not_found", "Run not found.")
			return
		}
		WriteError(w, http.StatusInternalServerError, "get_failed", "Could not fetch parent run.")
		return
	}

	var payload map[string]any
	if len(parent.TriggerPayload) > 0 {
		_ = json.Unmarshal(parent.TriggerPayload, &payload)
	}
	if payload == nil {
		payload = map[string]any{}
	}

	newID, err := d.Engine.RunFromReplay(r.Context(), parent.WorkflowID, parent.ID, payload)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "replay_failed", "Could not start replay.")
		return
	}
	WriteJSON(w, http.StatusCreated, map[string]any{"run_id": newID})
}

// parseIntDefault clamps an int query param to [1, max] with a fallback
// for missing/invalid input. Centralised so list+log handlers share
// identical semantics.
func parseIntDefault(s string, def, max int) int {
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 1 {
		return def
	}
	if n > max {
		return max
	}
	return n
}
