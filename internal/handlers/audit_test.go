package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/puppe1990/cais/pkg/cais"
	"github.com/puppe1990/cais/pkg/cais/middleware"

	"github.com/puppe1990/kiwify_dashboard/internal/store"
)

func newAuditHandler(t *testing.T) (*AuditHandler, store.Store) {
	t.Helper()
	s := setupTestStore(t)
	h := NewAuditHandler(s, testSite(), cais.Config{}, setupTestInertia(t))
	return h, s
}

func TestAuditList_RendersFromStore(t *testing.T) {
	h, s := newAuditHandler(t)

	if _, err := s.InsertAuditLog(store.AuditLog{
		UserID:         1,
		Action:         "sales.refund",
		ResourceType:   "sale",
		ResourceID:     "ord-1",
		RequestSummary: `{"id":"ord-1"}`,
		ResponseStatus: 200,
		ResponseBody:   `{"ok":true}`,
		IP:             "127.0.0.1",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.InsertAuditLog(store.AuditLog{
		UserID:         1,
		Action:         "finance.payout",
		ResourceType:   "payout",
		ResourceID:     "pay-1",
		RequestSummary: `{"amount":1000}`,
		ResponseStatus: 400,
		ResponseBody:   `{"error":"insufficient"}`,
		IP:             "10.0.0.1",
	}); err != nil {
		t.Fatal(err)
	}

	req := inertiaRequest(http.MethodGet, "/audit", nil)
	rr := httptest.NewRecorder()
	h.List(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rr.Code, rr.Body.String())
	}
	assertInertiaComponent(t, rr, "Audit")
	logs := assertInertiaProp(t, rr, "logs").([]any)
	if len(logs) != 2 {
		t.Fatalf("logs len = %d", len(logs))
	}
	// Newest first
	first := logs[0].(map[string]any)
	if first["action"] != "finance.payout" {
		t.Fatalf("first action = %v", first["action"])
	}
	if first["resourceType"] != "payout" {
		t.Fatalf("resourceType = %v", first["resourceType"])
	}
	if first["responseStatus"] != float64(400) {
		t.Fatalf("responseStatus = %v", first["responseStatus"])
	}
	if first["requestSummary"] != `{"amount":1000}` {
		t.Fatalf("summary = %v", first["requestSummary"])
	}
	if first["userId"] != float64(1) {
		t.Fatalf("userId = %v", first["userId"])
	}
	if _, ok := first["createdAt"]; !ok {
		t.Fatal("expected createdAt")
	}
	pag := assertInertiaProp(t, rr, "pagination").(map[string]any)
	if v, ok := pag["page"].(float64); !ok || v != 1 {
		if pag["page"] != 1 {
			t.Fatalf("pagination = %v", pag)
		}
	}
}

func TestAuditList_Empty(t *testing.T) {
	h, _ := newAuditHandler(t)

	req := inertiaRequest(http.MethodGet, "/audit", nil)
	rr := httptest.NewRecorder()
	h.List(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d", rr.Code)
	}
	assertInertiaComponent(t, rr, "Audit")
	logs := assertInertiaProp(t, rr, "logs").([]any)
	if len(logs) != 0 {
		t.Fatalf("logs len = %d, want 0", len(logs))
	}
}

func TestAuditList_AuthRedirect(t *testing.T) {
	h, _ := newAuditHandler(t)

	handler := middleware.RequireAuthFunc("/login", h.List)
	req := httptest.NewRequest(http.MethodGet, "/audit", nil)
	rr := httptest.NewRecorder()
	handler(rr, req)

	if rr.Code != http.StatusSeeOther || rr.Header().Get("Location") != "/login" {
		t.Fatalf("status=%d loc=%s", rr.Code, rr.Header().Get("Location"))
	}
}
