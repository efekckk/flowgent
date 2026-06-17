package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/efekckk/flowgent/internal/runlog"
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
	wfID := chi.URLParam(r, "id")
	if wfID == "" {
		WriteError(w, http.StatusBadRequest, "invalid_input", "Workflow id is required.")
		return
	}
	if _, ok := d.loadOwnedWorkflow(w, r, wfID); !ok {
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
	id := chi.URLParam(r, "id")
	if _, ok := d.loadOwnedRun(w, r, id); !ok {
		return
	}
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
	if d.RunLogs == nil {
		WriteError(w, http.StatusServiceUnavailable, "logs_disabled", "Run logs are not enabled.")
		return
	}
	id := chi.URLParam(r, "id")
	if _, ok := d.loadOwnedRun(w, r, id); !ok {
		return
	}
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

// handleStreamRun opens a Server-Sent Events stream that tails the run's
// log lines in real time. The handler first replays past rows from
// storage starting at ?since=<last_id> so a reconnecting client doesn't
// miss anything, then subscribes to the in-process Streamer for live
// events. A 30s heartbeat keeps the connection healthy through proxies.
func (d *Deps) handleStreamRun(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if _, ok := d.loadOwnedRun(w, r, id); !ok {
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	if d.Streamer == nil {
		http.Error(w, "streamer disabled", http.StatusServiceUnavailable)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // disable proxy buffering

	// Subscribe before draining storage so any event emitted during the
	// catch-up window lands in the channel rather than being missed.
	ch, unsub := d.Streamer.Subscribe(id)
	defer unsub()

	// Catch-up from DB so reconnecting clients don't miss past events.
	if d.RunLogs != nil {
		since, _ := strconv.ParseInt(r.URL.Query().Get("since"), 10, 64)
		if past, err := d.RunLogs.ListByRun(r.Context(), id, since, 1000); err == nil {
			for _, l := range past {
				writeSSEFrame(w, "log", logToEvent(l))
			}
			flusher.Flush()
		}
	}

	heartbeat := time.NewTicker(30 * time.Second)
	defer heartbeat.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case e, ok := <-ch:
			if !ok {
				return
			}
			writeSSEFrame(w, "log", e)
			flusher.Flush()
		case <-heartbeat.C:
			_, _ = io.WriteString(w, ": heartbeat\n\n")
			flusher.Flush()
		}
	}
}

// writeSSEFrame serialises one SSE message in the `event: <name>\ndata: <json>\n\n`
// shape that browser EventSource expects. Errors marshalling are dropped on
// purpose — a malformed event must never break the rest of the stream.
func writeSSEFrame(w io.Writer, event string, data any) {
	payload, err := json.Marshal(data)
	if err != nil {
		return
	}
	fmt.Fprintf(w, "event: %s\n", event)
	fmt.Fprintf(w, "data: %s\n\n", payload)
}

func logToEvent(l storage.RunLog) runlog.Event {
	return runlog.Event{
		RunID:   l.RunID,
		NodeID:  l.NodeID,
		Level:   l.Level,
		Message: l.Message,
		At:      l.At.Format(time.RFC3339Nano),
	}
}

// handleReplayRun forks a brand-new run that inherits the parent's
// trigger_payload. The engine's RunFromReplay owns the workflow lookup +
// persistence — this handler is just the HTTP front door.
func (d *Deps) handleReplayRun(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if _, ok := d.loadOwnedRun(w, r, id); !ok {
		return
	}
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
