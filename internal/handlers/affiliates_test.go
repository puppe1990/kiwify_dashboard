package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/puppe1990/cais/pkg/cais"
	"github.com/puppe1990/cais/pkg/cais/middleware"
	"github.com/puppe1990/cais/pkg/cais/session"

	"github.com/puppe1990/kiwify_dashboard/internal/kiwify"
	"github.com/puppe1990/kiwify_dashboard/internal/store"
)

type fakeAffiliatesAPI struct {
	listFn   func(ctx context.Context, q kiwify.AffiliatesQuery) (kiwify.AffiliatesPage, error)
	getFn    func(ctx context.Context, id string) (kiwify.Affiliate, error)
	updateFn func(ctx context.Context, id string, body map[string]any) (kiwify.Affiliate, error)

	lastQuery  kiwify.AffiliatesQuery
	updateID   string
	updateBody map[string]any
	listN      int
	getN       int
	updateN    int
}

func (f *fakeAffiliatesAPI) ListAffiliates(ctx context.Context, q kiwify.AffiliatesQuery) (kiwify.AffiliatesPage, error) {
	f.listN++
	f.lastQuery = q
	if f.listFn != nil {
		return f.listFn(ctx, q)
	}
	return kiwify.AffiliatesPage{Data: []kiwify.Affiliate{}}, nil
}

func (f *fakeAffiliatesAPI) GetAffiliate(ctx context.Context, id string) (kiwify.Affiliate, error) {
	f.getN++
	if f.getFn != nil {
		return f.getFn(ctx, id)
	}
	return kiwify.Affiliate{AffiliateID: id, Name: "Aff", Status: "active"}, nil
}

func (f *fakeAffiliatesAPI) UpdateAffiliate(ctx context.Context, id string, body map[string]any) (kiwify.Affiliate, error) {
	f.updateN++
	f.updateID = id
	f.updateBody = body
	if f.updateFn != nil {
		return f.updateFn(ctx, id, body)
	}
	return kiwify.Affiliate{AffiliateID: id, Status: "active", Commission: 100}, nil
}

func newAffiliatesHandler(t *testing.T, api AffiliatesAPI) (*AffiliatesHandler, store.Store) {
	t.Helper()
	s := setupTestStore(t)
	h := NewAffiliatesHandler(s, testAppSecret(), testSite(), cais.Config{}, setupTestInertia(t), api)
	return h, s
}

func TestAffiliatesList_Renders(t *testing.T) {
	fake := &fakeAffiliatesAPI{
		listFn: func(ctx context.Context, q kiwify.AffiliatesQuery) (kiwify.AffiliatesPage, error) {
			return kiwify.AffiliatesPage{
				Data: []kiwify.Affiliate{
					{AffiliateID: "aff-1", Name: "Maria", Email: "m@ex.com", Status: "active", Commission: 4600},
				},
				Pagination: kiwify.Pagination{Count: 1, PageNumber: 1, PageSize: 50},
			}, nil
		},
	}
	h, _ := newAffiliatesHandler(t, fake)

	req := inertiaRequest(http.MethodGet, "/affiliates?status=active&search=maria", nil)
	rr := httptest.NewRecorder()
	h.List(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rr.Code, rr.Body.String())
	}
	assertInertiaComponent(t, rr, "Affiliates")
	affs := assertInertiaProp(t, rr, "affiliates").([]any)
	if len(affs) != 1 {
		t.Fatalf("affiliates len = %d", len(affs))
	}
	a := affs[0].(map[string]any)
	if a["affiliate_id"] != "aff-1" || a["name"] != "Maria" {
		t.Fatalf("affiliate = %v", a)
	}
	if fake.lastQuery.Status != "active" || fake.lastQuery.Search != "maria" {
		t.Fatalf("query = %+v", fake.lastQuery)
	}
	if fake.lastQuery.PageSize != "50" {
		t.Fatalf("page size = %q", fake.lastQuery.PageSize)
	}
}

func TestAffiliatesShow_Renders(t *testing.T) {
	fake := &fakeAffiliatesAPI{
		getFn: func(ctx context.Context, id string) (kiwify.Affiliate, error) {
			return kiwify.Affiliate{
				AffiliateID: id,
				Name:        "João",
				Email:       "j@ex.com",
				Status:      "active",
				Commission:  1200,
			}, nil
		},
	}
	h, _ := newAffiliatesHandler(t, fake)

	req := inertiaRequest(http.MethodGet, "/affiliates/aff-99", nil)
	rr := httptest.NewRecorder()
	h.Show(rr, req, "aff-99")

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d", rr.Code)
	}
	assertInertiaComponent(t, rr, "AffiliateShow")
	aff := assertInertiaProp(t, rr, "affiliate").(map[string]any)
	if aff["affiliate_id"] != "aff-99" || aff["name"] != "João" {
		t.Fatalf("affiliate = %v", aff)
	}
}

func TestAffiliatesUpdate_SuccessWritesAudit(t *testing.T) {
	fake := &fakeAffiliatesAPI{}
	h, s := newAffiliatesHandler(t, fake)

	form := url.Values{
		"status":     {"blocked"},
		"commission": {"4600"},
	}
	req := inertiaRequest(http.MethodPost, "/affiliates/aff-1", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = session.WithUserID(req, 42)
	req.RemoteAddr = "203.0.113.10:9999"
	rr := httptest.NewRecorder()
	h.Update(rr, req, "aff-1")

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("status = %d body=%s", rr.Code, rr.Body.String())
	}
	if loc := rr.Header().Get("Location"); loc != "/affiliates/aff-1" {
		t.Fatalf("Location = %q", loc)
	}
	if fake.updateN != 1 || fake.updateID != "aff-1" {
		t.Fatalf("update call: n=%d id=%s", fake.updateN, fake.updateID)
	}
	if fake.updateBody["status"] != "blocked" {
		t.Fatalf("body = %v", fake.updateBody)
	}
	if fake.updateBody["commission"] != float64(4600) {
		t.Fatalf("commission = %v", fake.updateBody["commission"])
	}

	logs, err := s.ListAuditLogs(10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(logs) != 1 {
		t.Fatalf("audit logs = %d", len(logs))
	}
	log := logs[0]
	if log.Action != "affiliates.edit" || log.ResourceType != "affiliate" || log.ResourceID != "aff-1" {
		t.Fatalf("audit = %+v", log)
	}
	if log.ResponseStatus != 200 {
		t.Fatalf("ResponseStatus = %d", log.ResponseStatus)
	}
	if log.UserID != 42 {
		t.Fatalf("UserID = %d", log.UserID)
	}
	if log.IP != "203.0.113.10" {
		t.Fatalf("IP = %q", log.IP)
	}
	if !strings.Contains(log.RequestSummary, `"affiliate_id":"aff-1"`) {
		t.Fatalf("RequestSummary = %s", log.RequestSummary)
	}
}

func TestAffiliatesUpdate_APIErrorStillWritesAudit(t *testing.T) {
	fake := &fakeAffiliatesAPI{
		updateFn: func(ctx context.Context, id string, body map[string]any) (kiwify.Affiliate, error) {
			return kiwify.Affiliate{}, &kiwify.APIError{
				Status:      400,
				Message:     "invalid status",
				UserMessage: "Status inválido.",
				Body:        `{"error":"invalid status"}`,
			}
		},
	}
	h, s := newAffiliatesHandler(t, fake)

	form := url.Values{"status": {"blocked"}}
	req := inertiaRequest(http.MethodPost, "/affiliates/aff-2", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = session.WithUserID(req, 7)
	rr := httptest.NewRecorder()
	h.Update(rr, req, "aff-2")

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("status = %d", rr.Code)
	}
	logs, err := s.ListAuditLogs(10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(logs) != 1 {
		t.Fatalf("audit logs = %d, want 1 even on error", len(logs))
	}
	if logs[0].Action != "affiliates.edit" {
		t.Fatalf("action = %s", logs[0].Action)
	}
	if logs[0].ResponseStatus != 400 {
		t.Fatalf("ResponseStatus = %d", logs[0].ResponseStatus)
	}
	if !strings.Contains(logs[0].ResponseBody, "invalid status") {
		t.Fatalf("ResponseBody = %s", logs[0].ResponseBody)
	}
}

func TestAffiliatesUpdate_GenericErrorWritesAudit500(t *testing.T) {
	fake := &fakeAffiliatesAPI{
		updateFn: func(ctx context.Context, id string, body map[string]any) (kiwify.Affiliate, error) {
			return kiwify.Affiliate{}, errors.New("network down")
		},
	}
	h, s := newAffiliatesHandler(t, fake)

	form := url.Values{"status": {"active"}}
	req := inertiaRequest(http.MethodPost, "/affiliates/aff-3", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = session.WithUserID(req, 1)
	rr := httptest.NewRecorder()
	h.Update(rr, req, "aff-3")

	logs, err := s.ListAuditLogs(5, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(logs) != 1 || logs[0].ResponseStatus != 500 {
		t.Fatalf("logs = %+v", logs)
	}
}

func TestAffiliatesUpdate_EmptyBodyNoAPICall(t *testing.T) {
	fake := &fakeAffiliatesAPI{}
	h, s := newAffiliatesHandler(t, fake)

	req := inertiaRequest(http.MethodPost, "/affiliates/aff-1", strings.NewReader(""))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = session.WithUserID(req, 1)
	rr := httptest.NewRecorder()
	h.Update(rr, req, "aff-1")

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("status = %d", rr.Code)
	}
	if fake.updateN != 0 {
		t.Fatalf("UpdateAffiliate should not be called with empty body")
	}
	logs, err := s.ListAuditLogs(5, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(logs) != 0 {
		t.Fatalf("no audit for client validation, got %d", len(logs))
	}
}

func TestAffiliatesList_RequiresAuth(t *testing.T) {
	fake := &fakeAffiliatesAPI{}
	h, _ := newAffiliatesHandler(t, fake)
	handler := middleware.RequireAuthFunc("/login", h.List)

	req := httptest.NewRequest(http.MethodGet, "/affiliates", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusSeeOther || rr.Header().Get("Location") != "/login" {
		t.Fatalf("status=%d loc=%s", rr.Code, rr.Header().Get("Location"))
	}
	if fake.listN != 0 {
		t.Fatalf("ListAffiliates should not be called when unauthenticated")
	}
}

func TestAffiliatesUpdate_RequiresAuth(t *testing.T) {
	fake := &fakeAffiliatesAPI{}
	h, _ := newAffiliatesHandler(t, fake)
	handler := middleware.RequireAuthFunc("/login", cais.StringParam("id", h.Update))

	req := httptest.NewRequest(http.MethodPost, "/affiliates/aff-1", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusSeeOther || rr.Header().Get("Location") != "/login" {
		t.Fatalf("status=%d loc=%s", rr.Code, rr.Header().Get("Location"))
	}
	if fake.updateN != 0 {
		t.Fatalf("UpdateAffiliate should not be called when unauthenticated")
	}
}
