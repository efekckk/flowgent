package webhook

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

const maxBodyBytes = 1 << 20 // 1 MiB

// WebhookTrigger is the minimal info the handler needs to authenticate and
// route a request.
type WebhookTrigger struct {
	ID         string
	WorkflowID string
	Token      string
	Secret     []byte // nil/empty = no HMAC required
}

// Resolver hands the handler a trigger record by id. Returns ok=false on
// "trigger does not exist or is disabled".
type Resolver interface {
	ResolveWebhook(ctx context.Context, id string) (WebhookTrigger, bool, error)
}

// Firer dispatches an incoming webhook into the workflow engine. The
// signature mirrors scheduler.Firer; production wires both packages to the
// same impl.
type Firer interface {
	FireTrigger(ctx context.Context, triggerID, workflowID string, payload map[string]any) error
}

// Handler serves POST /webhooks/{trigger_id}/{token}. It does not own its
// route registration; mount with `r.Post("/webhooks/{trigger}/{token}", h)`
// or attach to a sub-router of the existing chi tree.
type Handler struct {
	res  Resolver
	fire Firer
}

func NewHandler(res Resolver, fire Firer) *Handler {
	return &Handler{res: res, fire: fire}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Path: /webhooks/{trigger_id}/{token}
	trimmed := strings.Trim(strings.TrimPrefix(r.URL.Path, "/webhooks/"), "/")
	parts := strings.Split(trimmed, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	triggerID, token := parts[0], parts[1]

	trg, ok, err := h.res.ResolveWebhook(r.Context(), triggerID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if !ok {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if !constantTimeStringEq(trg.Token, token) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxBodyBytes))
	if err != nil {
		// MaxBytesReader sets the response code via its own writer; if we
		// still need to send a status, 413 is the right call.
		http.Error(w, "payload too large", http.StatusRequestEntityTooLarge)
		return
	}

	if len(trg.Secret) > 0 {
		sig := r.Header.Get("X-Flowgent-Signature")
		if !VerifySignature(body, trg.Secret, sig) {
			http.Error(w, "signature mismatch", http.StatusUnauthorized)
			return
		}
	}

	payload := decodePayload(body, r.Header.Get("Content-Type"))
	payload["__headers"] = collectHeaders(r.Header)
	payload["__method"] = r.Method

	if err := h.fire.FireTrigger(r.Context(), trg.ID, trg.WorkflowID, payload); err != nil {
		http.Error(w, "dispatch failed", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_, _ = w.Write([]byte(`{"accepted":true}`))
}

// constantTimeStringEq is a constant-time comparison over two strings of
// equal length. We use it for the URL token even though the URL is logged
// by reverse proxies — the token is the only authn factor when no HMAC
// secret is set, so leaking timing about it is worth avoiding.
func constantTimeStringEq(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	var diff byte
	for i := 0; i < len(a); i++ {
		diff |= a[i] ^ b[i]
	}
	return diff == 0
}

func decodePayload(body []byte, contentType string) map[string]any {
	if strings.HasPrefix(strings.ToLower(contentType), "application/json") {
		var parsed map[string]any
		if err := json.Unmarshal(body, &parsed); err == nil && parsed != nil {
			return parsed
		}
	}
	// Fallback: wrap raw body so workflows can still expression-access it.
	return map[string]any{"body": string(body)}
}

func collectHeaders(h http.Header) map[string]any {
	out := make(map[string]any, len(h))
	for k, v := range h {
		if len(v) == 1 {
			out[k] = v[0]
		} else {
			out[k] = v
		}
	}
	return out
}
