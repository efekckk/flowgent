package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/efekckk/flowgent/internal/agent"
	"github.com/efekckk/flowgent/internal/idgen"
	"github.com/efekckk/flowgent/internal/provider"
	"github.com/efekckk/flowgent/internal/storage"
)

type chatRequest struct {
	Message string `json:"message"`
	Model   string `json:"model"`
}

type sseEvent struct {
	Type    string          `json:"type"`
	Content string          `json:"content,omitempty"`
	Tool    string          `json:"tool,omitempty"`
	Payload json.RawMessage `json:"payload,omitempty"`
	Error   string          `json:"error,omitempty"`
}

func (d *Deps) handleChat(w http.ResponseWriter, r *http.Request) {
	wfID := chi.URLParam(r, "id")
	u, ok := userFromContext(r.Context())
	if !ok {
		WriteError(w, http.StatusUnauthorized, "no_session", "Authentication required.")
		return
	}
	var req chatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid_json", "Body must be JSON.")
		return
	}
	if req.Message == "" {
		WriteError(w, http.StatusBadRequest, "missing_message", "Message is required.")
		return
	}
	if req.Model == "" {
		req.Model = "gpt-4o"
	}

	wf, err := d.Workflows.Get(r.Context(), wfID)
	if err != nil {
		WriteError(w, http.StatusNotFound, "not_found", "Workflow not found.")
		return
	}
	ver, err := d.Workflows.GetVersion(r.Context(), wf.ID, wf.CurrentVersion)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "version_get_failed", "Could not fetch version.")
		return
	}

	thr, err := d.ensureThread(r, wf.ID, u.ID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "thread_failed", "Could not start chat thread.")
		return
	}

	history, err := d.loadHistory(r, thr.ID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "history_failed", "Could not load chat history.")
		return
	}

	_ = d.ChatMessages.Insert(r.Context(), storage.ChatMessage{
		ID: idgen.NewChatMessage(), ThreadID: thr.ID, Role: "user", Content: req.Message,
	})

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	flusher, _ := w.(http.Flusher)

	out, runErr := d.Agent.Run(r.Context(), agent.RunRequest{
		Model:           req.Model,
		UserMessage:     req.Message,
		History:         history,
		CurrentWorkflow: ver.Definition,
	})
	if runErr != nil {
		writeSSE(w, flusher, sseEvent{Type: "error", Error: runErr.Error()})
		return
	}

	if out.AssistantText != "" {
		writeSSE(w, flusher, sseEvent{Type: "text", Content: out.AssistantText})
	}
	switch out.ToolName {
	case "propose_workflow":
		writeSSE(w, flusher, sseEvent{Type: "proposal", Tool: out.ToolName, Payload: out.ProposedWorkflow})
		_ = d.ChatMessages.Insert(r.Context(), storage.ChatMessage{
			ID: idgen.NewChatMessage(), ThreadID: thr.ID, Role: "assistant",
			Content:   out.AssistantText,
			ToolCalls: mustJSON([]map[string]any{{"name": "propose_workflow", "args": json.RawMessage(out.ProposedWorkflow)}}),
			Model:     req.Model,
		})
	case "edit_workflow":
		writeSSE(w, flusher, sseEvent{Type: "patch", Tool: out.ToolName, Payload: out.Patches})
		_ = d.ChatMessages.Insert(r.Context(), storage.ChatMessage{
			ID: idgen.NewChatMessage(), ThreadID: thr.ID, Role: "assistant",
			Content:   out.AssistantText,
			ToolCalls: mustJSON([]map[string]any{{"name": "edit_workflow", "args": json.RawMessage(out.Patches)}}),
			Model:     req.Model,
		})
	default:
		_ = d.ChatMessages.Insert(r.Context(), storage.ChatMessage{
			ID: idgen.NewChatMessage(), ThreadID: thr.ID, Role: "assistant",
			Content: out.AssistantText, Model: req.Model,
		})
	}
	writeSSE(w, flusher, sseEvent{Type: "done"})
}

func (d *Deps) ensureThread(r *http.Request, workflowID, userID string) (storage.ChatThread, error) {
	thr, err := d.ChatThreads.GetByWorkflowAndUser(r.Context(), workflowID, userID)
	if err == nil {
		return thr, nil
	}
	if !errors.Is(err, storage.ErrNotFound) {
		return storage.ChatThread{}, err
	}
	thr = storage.ChatThread{
		ID:         idgen.NewChatThread(),
		WorkflowID: workflowID,
		UserID:     userID,
	}
	if err := d.ChatThreads.Insert(r.Context(), thr); err != nil {
		return storage.ChatThread{}, err
	}
	return thr, nil
}

func (d *Deps) loadHistory(r *http.Request, threadID string) ([]provider.Message, error) {
	rows, err := d.ChatMessages.ListByThread(r.Context(), threadID, 50)
	if err != nil {
		return nil, err
	}
	out := make([]provider.Message, 0, len(rows))
	for _, m := range rows {
		out = append(out, provider.Message{Role: m.Role, Content: m.Content})
	}
	return out, nil
}

func writeSSE(w http.ResponseWriter, flusher http.Flusher, ev sseEvent) {
	raw, _ := json.Marshal(ev)
	fmt.Fprintf(w, "data: %s\n\n", raw)
	if flusher != nil {
		flusher.Flush()
	}
}

func mustJSON(v any) json.RawMessage {
	b, _ := json.Marshal(v)
	return json.RawMessage(b)
}
