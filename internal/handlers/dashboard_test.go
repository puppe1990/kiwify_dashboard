package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/puppe1990/cais/pkg/cais"

	"github.com/puppe1990/kiwify_dashboard/internal/store"
)

func TestDashboardHandler_InertiaComponent(t *testing.T) {
	h := NewDashboardHandler(
		setupTestRenderer(t),
		setupTestStore(t),
		testAppSecret(),
		testSite(),
		cais.Config{},
		setupTestInertia(t),
	)

	req := inertiaRequest(http.MethodGet, "/dashboard", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want %d body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}
	assertInertiaComponent(t, rr, "Dashboard")
	// Without Kiwify credentials, client section error is expected.
	props := parseInertiaJSON(t, rr)["props"].(map[string]any)
	if props["dateRange"] == nil {
		t.Error("expected dateRange prop")
	}
	errs, _ := props["errors"].(map[string]any)
	if errs == nil || errs["client"] == nil {
		t.Errorf("expected client error when unconfigured, got errors=%v", props["errors"])
	}
}

func TestDashboardHandler_WithLocalEvents(t *testing.T) {
	s := setupTestStore(t)
	if _, err := s.InsertWebhookEvent(store.WebhookEvent{
		EventType:   "order_approved",
		PayloadJSON: `{}`,
		ProcessedOK: true,
	}); err != nil {
		t.Fatal(err)
	}

	h := NewDashboardHandler(
		setupTestRenderer(t),
		s,
		testAppSecret(),
		testSite(),
		cais.Config{},
		setupTestInertia(t),
	)

	req := inertiaRequest(http.MethodGet, "/dashboard", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d", rr.Code)
	}
	events := assertInertiaProp(t, rr, "events")
	list, ok := events.([]any)
	if !ok || len(list) != 1 {
		t.Fatalf("events = %v", events)
	}
	first, ok := list[0].(map[string]any)
	if !ok || first["eventType"] != "order_approved" {
		t.Fatalf("event = %v", first)
	}
}
