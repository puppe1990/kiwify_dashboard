package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/puppe1990/kiwify_dashboard/internal/store"
)

// header names stored with inbound webhook events (lower-case lookup).
var webhookHeaderAllowlist = []string{
	"Content-Type",
	"User-Agent",
	"X-Request-Id",
	"X-Forwarded-For",
	"X-Forwarded-Proto",
	"X-Kiwify-Event",
	"X-Kiwify-Signature",
	"X-Webhook-Token",
}

// KiwifyWebhookHandler receives public Kiwify webhook POSTs (no session auth).
type KiwifyWebhookHandler struct {
	store store.Store
}

// NewKiwifyWebhookHandler constructs a public receiver handler.
func NewKiwifyWebhookHandler(s store.Store) *KiwifyWebhookHandler {
	return &KiwifyWebhookHandler{store: s}
}

// Receive handles POST /webhooks/kiwify/{token}.
// Wrong/missing token → 404. Invalid JSON → 400. Success → 200 quickly after store insert.
func (h *KiwifyWebhookHandler) Receive(w http.ResponseWriter, r *http.Request, token string) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	settings, err := h.store.GetKiwifySettings()
	if err != nil || settings.WebhookReceiveToken == "" {
		http.NotFound(w, r)
		return
	}
	// Constant-time-ish compare without leaking length via early equality of empty.
	if !secureTokenEqual(settings.WebhookReceiveToken, token) {
		http.NotFound(w, r)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20)) // 1 MiB cap
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if len(body) == 0 {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	eventType := inferWebhookEventType(payload)
	headersJSON := selectedHeadersJSON(r.Header)

	payloadJSON := string(body)
	// Re-marshal for compact/canonical storage if original was valid map.
	if raw, err := json.Marshal(payload); err == nil {
		payloadJSON = string(raw)
	}

	if _, err := h.store.InsertWebhookEvent(store.WebhookEvent{
		EventType:   eventType,
		PayloadJSON: payloadJSON,
		HeadersJSON: headersJSON,
		ProcessedOK: true,
	}); err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"ok":true}`))
}

func secureTokenEqual(expected, got string) bool {
	if len(expected) != len(got) {
		return false
	}
	var v byte
	for i := 0; i < len(expected); i++ {
		v |= expected[i] ^ got[i]
	}
	return v == 0
}

// inferWebhookEventType picks a useful event label from common Kiwify payload keys.
func inferWebhookEventType(payload map[string]any) string {
	keys := []string{"order_status", "webhook_event_type", "trigger", "event", "type"}
	for _, k := range keys {
		if v, ok := payload[k]; ok {
			if s := stringifyEventValue(v); s != "" {
				return s
			}
		}
	}
	// Nested order.status sometimes used in sale payloads.
	if order, ok := payload["order"].(map[string]any); ok {
		if s := stringifyEventValue(order["status"]); s != "" {
			return s
		}
		if s := stringifyEventValue(order["order_status"]); s != "" {
			return s
		}
	}
	return "unknown"
}

func stringifyEventValue(v any) string {
	switch x := v.(type) {
	case string:
		return strings.TrimSpace(x)
	case float64:
		// avoid storing bare numbers as event types
		return ""
	case bool:
		return ""
	default:
		return ""
	}
}

func selectedHeadersJSON(h http.Header) string {
	out := map[string]string{}
	for _, name := range webhookHeaderAllowlist {
		if v := h.Get(name); v != "" {
			out[name] = v
		}
	}
	raw, err := json.Marshal(out)
	if err != nil {
		return "{}"
	}
	return string(raw)
}
