package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/puppe1990/kiwify_dashboard/internal/store"
)

func seedWebhookSettings(t *testing.T, s store.Store, token string) {
	t.Helper()
	if err := s.SaveKiwifySettings(store.KiwifySettings{
		AccountID: "acc", ClientID: "cid", ClientSecretCiphertext: "cipher",
		WebhookReceiveToken: token,
	}); err != nil {
		t.Fatal(err)
	}
}

func TestKiwifyWebhookReceive_ValidTokenStoresEvent(t *testing.T) {
	s := setupTestStore(t)
	seedWebhookSettings(t, s, "good-token-abcdef")
	h := NewKiwifyWebhookHandler(s)

	body := `{"order_status":"paid","order_id":"ord-1","amount":100}`
	req := httptest.NewRequest(http.MethodPost, "/webhooks/kiwify/good-token-abcdef", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Kiwify-Webhook/1.0")
	rr := httptest.NewRecorder()
	h.Receive(rr, req, "good-token-abcdef")

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rr.Code, rr.Body.String())
	}

	events, err := s.ListWebhookEvents(10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 {
		t.Fatalf("events len = %d", len(events))
	}
	ev := events[0]
	if ev.EventType != "paid" {
		t.Fatalf("event_type = %q, want paid", ev.EventType)
	}
	if !ev.ProcessedOK {
		t.Fatal("expected processed_ok")
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(ev.PayloadJSON), &payload); err != nil {
		t.Fatal(err)
	}
	if payload["order_id"] != "ord-1" {
		t.Fatalf("payload = %v", payload)
	}
	var headers map[string]string
	if err := json.Unmarshal([]byte(ev.HeadersJSON), &headers); err != nil {
		t.Fatal(err)
	}
	if headers["Content-Type"] != "application/json" {
		t.Fatalf("headers = %v", headers)
	}
	if headers["User-Agent"] != "Kiwify-Webhook/1.0" {
		t.Fatalf("headers = %v", headers)
	}
}

func TestKiwifyWebhookReceive_WrongToken404(t *testing.T) {
	s := setupTestStore(t)
	seedWebhookSettings(t, s, "good-token")
	h := NewKiwifyWebhookHandler(s)

	req := httptest.NewRequest(http.MethodPost, "/webhooks/kiwify/wrong", strings.NewReader(`{"a":1}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.Receive(rr, req, "wrong")

	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rr.Code)
	}
	events, err := s.ListWebhookEvents(10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 0 {
		t.Fatalf("should not store event on wrong token, got %d", len(events))
	}
}

func TestKiwifyWebhookReceive_InvalidJSON400(t *testing.T) {
	s := setupTestStore(t)
	seedWebhookSettings(t, s, "tok")
	h := NewKiwifyWebhookHandler(s)

	req := httptest.NewRequest(http.MethodPost, "/webhooks/kiwify/tok", strings.NewReader(`not-json`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.Receive(rr, req, "tok")

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rr.Code)
	}
	events, err := s.ListWebhookEvents(10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 0 {
		t.Fatalf("should not store invalid json, got %d", len(events))
	}
}

func TestKiwifyWebhookReceive_EmptyBody400(t *testing.T) {
	s := setupTestStore(t)
	seedWebhookSettings(t, s, "tok")
	h := NewKiwifyWebhookHandler(s)

	req := httptest.NewRequest(http.MethodPost, "/webhooks/kiwify/tok", strings.NewReader(""))
	rr := httptest.NewRecorder()
	h.Receive(rr, req, "tok")

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rr.Code)
	}
}

func TestInferWebhookEventType(t *testing.T) {
	cases := []struct {
		name string
		in   map[string]any
		want string
	}{
		{"order_status", map[string]any{"order_status": "refunded"}, "refunded"},
		{"webhook_event_type", map[string]any{"webhook_event_type": "compra_aprovada"}, "compra_aprovada"},
		{"trigger", map[string]any{"trigger": "pix_gerado"}, "pix_gerado"},
		{"event", map[string]any{"event": "chargeback"}, "chargeback"},
		{"type", map[string]any{"type": "subscription_late"}, "subscription_late"},
		{"nested order.status", map[string]any{"order": map[string]any{"status": "paid"}}, "paid"},
		{"unknown", map[string]any{"foo": "bar"}, "unknown"},
		{"prefer order_status", map[string]any{"order_status": "paid", "type": "other"}, "paid"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := inferWebhookEventType(tc.in)
			if got != tc.want {
				t.Fatalf("got %q want %q", got, tc.want)
			}
		})
	}
}
