package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/puppe1990/cais/pkg/cais"

	"github.com/puppe1990/kiwify_dashboard/internal/store"
)

func TestEventsList_RendersPaginated(t *testing.T) {
	s := setupTestStore(t)
	h := NewEventsHandler(s, testSite(), cais.Config{}, setupTestInertia(t))

	for i := 0; i < 3; i++ {
		if _, err := s.InsertWebhookEvent(store.WebhookEvent{
			EventType:   "paid",
			PayloadJSON: `{"i":1}`,
			HeadersJSON: `{}`,
			ProcessedOK: true,
		}); err != nil {
			t.Fatal(err)
		}
	}

	req := inertiaRequest(http.MethodGet, "/events", nil)
	rr := httptest.NewRecorder()
	h.List(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rr.Code, rr.Body.String())
	}
	assertInertiaComponent(t, rr, "Events")
	events := assertInertiaProp(t, rr, "events").([]any)
	if len(events) != 3 {
		t.Fatalf("events len = %d", len(events))
	}
	ev := events[0].(map[string]any)
	if ev["eventType"] != "paid" {
		t.Fatalf("event = %v", ev)
	}
	if _, ok := ev["payloadJson"]; !ok {
		t.Fatal("expected payloadJson on event props")
	}
	pag := assertInertiaProp(t, rr, "pagination").(map[string]any)
	if pag["page"] != float64(1) && pag["page"] != 1 {
		// JSON numbers decode as float64
		if v, ok := pag["page"].(float64); !ok || v != 1 {
			t.Fatalf("pagination = %v", pag)
		}
	}
}
