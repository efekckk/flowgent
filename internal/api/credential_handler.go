package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/efekckk/flowgent/internal/crypto"
	"github.com/efekckk/flowgent/internal/idgen"
	"github.com/efekckk/flowgent/internal/storage"
)

type createCredentialRequest struct {
	Name   string         `json:"name"`
	Type   string         `json:"type"`
	Secret map[string]any `json:"secret"`
	Meta   map[string]any `json:"meta"`
}

type credentialDTO struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Type      string    `json:"type"`
	CreatedAt time.Time `json:"created_at"`
}

type credentialList struct {
	Items []credentialDTO `json:"items"`
}

func (d *Deps) handleCreateCredential(w http.ResponseWriter, r *http.Request) {
	u, ok := userFromContext(r.Context())
	if !ok {
		WriteError(w, http.StatusUnauthorized, "no_session", "Authentication required.")
		return
	}
	var req createCredentialRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid_json", "Body must be JSON.")
		return
	}
	if req.Name == "" || req.Type == "" {
		WriteError(w, http.StatusBadRequest, "invalid_input", "name and type are required.")
		return
	}
	if req.Secret == nil {
		req.Secret = map[string]any{}
	}
	plain, err := json.Marshal(req.Secret)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "invalid_secret", "Secret is not JSON-serialisable.")
		return
	}
	encrypted, err := crypto.Encrypt(plain, d.CredentialKey)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "encrypt_failed", "Could not encrypt secret.")
		return
	}
	metaBytes, _ := json.Marshal(req.Meta)
	if len(metaBytes) == 0 {
		metaBytes = []byte("{}")
	}

	workspaces, err := d.Workspaces.FindByOwner(r.Context(), u.ID)
	if err != nil || len(workspaces) == 0 {
		WriteError(w, http.StatusInternalServerError, "workspace_missing", "No workspace.")
		return
	}
	cred := storage.Credential{
		ID:          idgen.NewCredential(),
		WorkspaceID: workspaces[0].ID,
		Name:        req.Name,
		Type:        req.Type,
		Encrypted:   encrypted,
		Meta:        metaBytes,
	}
	if err := d.Credentials.Insert(r.Context(), cred); err != nil {
		if errors.Is(err, storage.ErrConflict) {
			WriteError(w, http.StatusConflict, "name_taken", "A credential with that name already exists.")
			return
		}
		WriteError(w, http.StatusInternalServerError, "insert_failed", "Could not save credential.")
		return
	}
	WriteJSON(w, http.StatusCreated, credentialDTO{
		ID: cred.ID, Name: cred.Name, Type: cred.Type, CreatedAt: time.Now().UTC(),
	})
}

func (d *Deps) handleListCredentials(w http.ResponseWriter, r *http.Request) {
	u, ok := userFromContext(r.Context())
	if !ok {
		WriteError(w, http.StatusUnauthorized, "no_session", "Authentication required.")
		return
	}
	workspaces, err := d.Workspaces.FindByOwner(r.Context(), u.ID)
	if err != nil || len(workspaces) == 0 {
		WriteJSON(w, http.StatusOK, credentialList{Items: []credentialDTO{}})
		return
	}
	rows, err := d.Credentials.ListByWorkspace(r.Context(), workspaces[0].ID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "list_failed", "Could not list credentials.")
		return
	}
	out := credentialList{Items: make([]credentialDTO, 0, len(rows))}
	for _, c := range rows {
		out.Items = append(out.Items, credentialDTO{
			ID: c.ID, Name: c.Name, Type: c.Type, CreatedAt: c.CreatedAt,
		})
	}
	WriteJSON(w, http.StatusOK, out)
}

func (d *Deps) handleDeleteCredential(w http.ResponseWriter, r *http.Request) {
	u, ok := userFromContext(r.Context())
	if !ok {
		WriteError(w, http.StatusUnauthorized, "no_session", "Authentication required.")
		return
	}
	id := chi.URLParam(r, "id")
	cred, err := d.Credentials.Get(r.Context(), id)
	if err != nil {
		WriteError(w, http.StatusNotFound, "not_found", "Credential not found.")
		return
	}
	workspaces, _ := d.Workspaces.FindByOwner(r.Context(), u.ID)
	owned := false
	for _, ws := range workspaces {
		if ws.ID == cred.WorkspaceID {
			owned = true
			break
		}
	}
	if !owned {
		WriteError(w, http.StatusNotFound, "not_found", "Credential not found.")
		return
	}
	if err := d.Credentials.Delete(r.Context(), id); err != nil {
		WriteError(w, http.StatusInternalServerError, "delete_failed", "Could not delete.")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
