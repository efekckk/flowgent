package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/robfig/cron/v3"

	"github.com/efekckk/flowgent/internal/idgen"
	"github.com/efekckk/flowgent/internal/storage"
)

// cronParser mirrors the parser the scheduler uses, so a malformed expression
// fails fast at create time with 400 rather than silently disappearing into
// the table only to be skipped at boot.
var cronParser = cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)

type triggerCreateRequest struct {
	Kind   string         `json:"kind"`
	Config map[string]any `json:"config"`
}

type triggerUpdateRequest struct {
	Config  map[string]any `json:"config"`
	Enabled *bool          `json:"enabled"`
}

type triggerDTO struct {
	ID         string         `json:"id"`
	WorkflowID string         `json:"workflow_id"`
	Kind       string         `json:"kind"`
	Config     map[string]any `json:"config"`
	Enabled    bool           `json:"enabled"`
	WebhookURL string         `json:"webhook_url,omitempty"`
}

type triggerListResponse struct {
	Items []triggerDTO `json:"items"`
}

func (d *Deps) handleCreateTrigger(w http.ResponseWriter, r *http.Request) {
	wfID := chi.URLParam(r, "id")
	if wfID == "" {
		WriteError(w, http.StatusBadRequest, "invalid_input", "Workflow id is required.")
		return
	}
	if _, ok := d.loadOwnedWorkflow(w, r, wfID); !ok {
		return
	}

	var req triggerCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid_json", "Body must be JSON.")
		return
	}
	if req.Config == nil {
		req.Config = map[string]any{}
	}
	if req.Kind != "cron" && req.Kind != "webhook" {
		WriteError(w, http.StatusBadRequest, "invalid_kind", "kind must be cron or webhook.")
		return
	}
	if req.Kind == "cron" {
		expr, _ := req.Config["cron"].(string)
		if _, err := cronParser.Parse(expr); err != nil {
			WriteError(w, http.StatusBadRequest, "invalid_cron", "Invalid cron expression: "+err.Error())
			return
		}
	}
	if req.Kind == "webhook" {
		if tok, _ := req.Config["token"].(string); tok == "" {
			req.Config["token"] = idgen.NewToken()
		}
	}

	cfgBytes, _ := json.Marshal(req.Config)
	t := storage.Trigger{
		ID:         idgen.NewTrigger(),
		WorkflowID: wfID,
		Kind:       req.Kind,
		Config:     cfgBytes,
		Enabled:    true,
	}
	if err := d.Triggers.Insert(r.Context(), t); err != nil {
		WriteError(w, http.StatusInternalServerError, "insert_failed", "Could not save trigger.")
		return
	}
	if req.Kind == "cron" && d.Scheduler != nil {
		expr, _ := req.Config["cron"].(string)
		_ = d.Scheduler.Add(t.ID, t.WorkflowID, expr)
	}

	out := triggerDTO{
		ID:         t.ID,
		WorkflowID: t.WorkflowID,
		Kind:       t.Kind,
		Config:     req.Config,
		Enabled:    true,
	}
	if t.Kind == "webhook" {
		tok, _ := req.Config["token"].(string)
		out.WebhookURL = webhookURL(d.PublicBaseURL, t.ID, tok)
	}
	WriteJSON(w, http.StatusCreated, out)
}

func (d *Deps) handleListTriggers(w http.ResponseWriter, r *http.Request) {
	wfID := chi.URLParam(r, "id")
	if _, ok := d.loadOwnedWorkflow(w, r, wfID); !ok {
		return
	}
	rows, err := d.Triggers.ListByWorkflow(r.Context(), wfID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "list_failed", "Could not list triggers.")
		return
	}
	out := triggerListResponse{Items: make([]triggerDTO, 0, len(rows))}
	for _, t := range rows {
		var cfg map[string]any
		_ = json.Unmarshal(t.Config, &cfg)
		dto := triggerDTO{
			ID:         t.ID,
			WorkflowID: t.WorkflowID,
			Kind:       t.Kind,
			Config:     cfg,
			Enabled:    t.Enabled,
		}
		if t.Kind == "webhook" {
			tok, _ := cfg["token"].(string)
			dto.WebhookURL = webhookURL(d.PublicBaseURL, t.ID, tok)
		}
		out.Items = append(out.Items, dto)
	}
	WriteJSON(w, http.StatusOK, out)
}

func (d *Deps) handleUpdateTrigger(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req triggerUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid_json", "Body must be JSON.")
		return
	}
	cur, ok := d.loadOwnedTrigger(w, r, id)
	if !ok {
		return
	}
	enabled := cur.Enabled
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	if req.Config == nil {
		// Preserve existing config when caller omits it.
		_ = json.Unmarshal(cur.Config, &req.Config)
		if req.Config == nil {
			req.Config = map[string]any{}
		}
	}
	if cur.Kind == "cron" {
		expr, _ := req.Config["cron"].(string)
		if _, err := cronParser.Parse(expr); err != nil {
			WriteError(w, http.StatusBadRequest, "invalid_cron", "Invalid cron expression: "+err.Error())
			return
		}
	}
	cfgBytes, _ := json.Marshal(req.Config)
	if err := d.Triggers.UpdateConfig(r.Context(), id, cfgBytes, enabled); err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			WriteError(w, http.StatusNotFound, "not_found", "Trigger not found.")
			return
		}
		WriteError(w, http.StatusInternalServerError, "update_failed", "Could not update trigger.")
		return
	}
	if cur.Kind == "cron" && d.Scheduler != nil {
		d.Scheduler.Remove(id)
		if enabled {
			expr, _ := req.Config["cron"].(string)
			_ = d.Scheduler.Add(id, cur.WorkflowID, expr)
		}
	}
	WriteJSON(w, http.StatusOK, triggerDTO{
		ID:         id,
		WorkflowID: cur.WorkflowID,
		Kind:       cur.Kind,
		Config:     req.Config,
		Enabled:    enabled,
	})
}

func (d *Deps) handleDeleteTrigger(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if _, ok := d.loadOwnedTrigger(w, r, id); !ok {
		return
	}
	if d.Scheduler != nil {
		d.Scheduler.Remove(id)
	}
	if err := d.Triggers.Delete(r.Context(), id); err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			WriteError(w, http.StatusNotFound, "not_found", "Trigger not found.")
			return
		}
		WriteError(w, http.StatusInternalServerError, "delete_failed", "Could not delete trigger.")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func webhookURL(base, triggerID, token string) string {
	return base + "/webhooks/" + triggerID + "/" + token
}
