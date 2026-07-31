package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/puppe1990/cais/pkg/cais"
	"github.com/puppe1990/cais/pkg/cais/middleware"
	"github.com/puppe1990/cais/pkg/cais/session"
	"github.com/puppe1990/kiwify_dashboard/internal/kiwify"
	"github.com/puppe1990/kiwify_dashboard/internal/store"
)

type fakeSalesAPI struct {
	listFn   func(ctx context.Context, q kiwify.SalesQuery) (kiwify.SalesPage, error)
	getFn    func(ctx context.Context, id string) (kiwify.Sale, error)
	refundFn func(ctx context.Context, id string, pixKey string) error

	lastQuery kiwify.SalesQuery
	refundID  string
	refundPix string
	refundN   int
}

func (f *fakeSalesAPI) ListSales(ctx context.Context, q kiwify.SalesQuery) (kiwify.SalesPage, error) {
	f.lastQuery = q
	if f.listFn != nil {
		return f.listFn(ctx, q)
	}
	return kiwify.SalesPage{Data: []kiwify.Sale{}}, nil
}

func (f *fakeSalesAPI) GetSale(ctx context.Context, id string) (kiwify.Sale, error) {
	if f.getFn != nil {
		return f.getFn(ctx, id)
	}
	return kiwify.Sale{ID: id, Reference: "REF", Status: "paid"}, nil
}

func (f *fakeSalesAPI) RefundSale(ctx context.Context, id string, pixKey string) error {
	f.refundN++
	f.refundID = id
	f.refundPix = pixKey
	if f.refundFn != nil {
		return f.refundFn(ctx, id, pixKey)
	}
	return nil
}

func newSalesHandler(t *testing.T, api SalesAPI) (*SalesHandler, store.Store) {
	t.Helper()
	s := setupTestStore(t)
	h := NewSalesHandler(s, testAppSecret(), testSite(), cais.Config{}, setupTestInertia(t), api)
	return h, s
}

func TestClampSalesDateRange_DefaultsLast30Days(t *testing.T) {
	now := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)
	start, end := clampSalesDateRange("", "", now)
	if end != "2024-06-15" {
		t.Fatalf("end = %s", end)
	}
	if start != "2024-05-16" {
		t.Fatalf("start = %s, want 2024-05-16 (30 days before)", start)
	}
}

func TestClampSalesDateRange_ClampsTo90Days(t *testing.T) {
	now := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)
	start, end := clampSalesDateRange("2024-01-01", "2024-06-15", now)
	if end != "2024-06-15" {
		t.Fatalf("end = %s", end)
	}
	// 90 days before 2024-06-15
	wantStart := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC).AddDate(0, 0, -90).Format("2006-01-02")
	if start != wantStart {
		t.Fatalf("start = %s, want %s (clamped to 90d)", start, wantStart)
	}
}

func TestClampSalesDateRange_AcceptsValidRange(t *testing.T) {
	now := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)
	start, end := clampSalesDateRange("2024-06-01", "2024-06-10", now)
	if start != "2024-06-01" || end != "2024-06-10" {
		t.Fatalf("got %s → %s", start, end)
	}
}

func TestSalesList_DefaultDateRangePassedToAPI(t *testing.T) {
	fake := &fakeSalesAPI{}
	h, _ := newSalesHandler(t, fake)

	req := inertiaRequest(http.MethodGet, "/sales", nil)
	rr := httptest.NewRecorder()
	h.List(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rr.Code, rr.Body.String())
	}
	assertInertiaComponent(t, rr, "Sales")

	// Relative to wall clock — just ensure both dates set and within 31 days.
	if fake.lastQuery.StartDate == "" || fake.lastQuery.EndDate == "" {
		t.Fatalf("query dates empty: %+v", fake.lastQuery)
	}
	start, err := time.Parse("2006-01-02", fake.lastQuery.StartDate)
	if err != nil {
		t.Fatal(err)
	}
	end, err := time.Parse("2006-01-02", fake.lastQuery.EndDate)
	if err != nil {
		t.Fatal(err)
	}
	days := int(end.Sub(start).Hours() / 24)
	if days < 29 || days > 31 {
		t.Fatalf("default range days = %d, want ~30", days)
	}
}

func TestSalesList_ClampsWideRange(t *testing.T) {
	fake := &fakeSalesAPI{}
	h, _ := newSalesHandler(t, fake)

	req := inertiaRequest(http.MethodGet, "/sales?start_date=2020-01-01&end_date=2024-06-15", nil)
	rr := httptest.NewRecorder()
	h.List(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d", rr.Code)
	}
	if fake.lastQuery.EndDate != "2024-06-15" {
		t.Fatalf("end = %s", fake.lastQuery.EndDate)
	}
	wantStart := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC).AddDate(0, 0, -90).Format("2006-01-02")
	if fake.lastQuery.StartDate != wantStart {
		t.Fatalf("start = %s, want clamped %s", fake.lastQuery.StartDate, wantStart)
	}
	dr := assertInertiaProp(t, rr, "dateRange").(map[string]any)
	if dr["start"] != wantStart || dr["end"] != "2024-06-15" {
		t.Fatalf("dateRange prop = %v", dr)
	}
}

func TestSalesShow_RendersSale(t *testing.T) {
	fake := &fakeSalesAPI{
		getFn: func(ctx context.Context, id string) (kiwify.Sale, error) {
			return kiwify.Sale{ID: id, Reference: "ABC", Status: "paid", NetAmount: 1500}, nil
		},
	}
	h, _ := newSalesHandler(t, fake)

	req := inertiaRequest(http.MethodGet, "/sales/sale-99", nil)
	rr := httptest.NewRecorder()
	h.Show(rr, req, "sale-99")

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d", rr.Code)
	}
	assertInertiaComponent(t, rr, "SaleShow")
	sale := assertInertiaProp(t, rr, "sale").(map[string]any)
	if sale["id"] != "sale-99" || sale["reference"] != "ABC" {
		t.Fatalf("sale = %v", sale)
	}
}

func TestSalesRefund_SuccessWritesAudit200(t *testing.T) {
	fake := &fakeSalesAPI{}
	h, s := newSalesHandler(t, fake)

	form := url.Values{"pixKey": {"chave-pix@ex.com"}}
	req := inertiaRequest(http.MethodPost, "/sales/ord1/refund", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = session.WithUserID(req, 42)
	req.RemoteAddr = "203.0.113.10:9999"
	rr := httptest.NewRecorder()
	h.Refund(rr, req, "ord1")

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303 body=%s", rr.Code, rr.Body.String())
	}
	if loc := rr.Header().Get("Location"); loc != "/sales/ord1" {
		t.Fatalf("Location = %q", loc)
	}
	if fake.refundN != 1 || fake.refundID != "ord1" || fake.refundPix != "chave-pix@ex.com" {
		t.Fatalf("refund call: n=%d id=%s pix=%s", fake.refundN, fake.refundID, fake.refundPix)
	}

	logs, err := s.ListAuditLogs(10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(logs) != 1 {
		t.Fatalf("audit logs = %d", len(logs))
	}
	log := logs[0]
	if log.Action != "sales.refund" || log.ResourceType != "sale" || log.ResourceID != "ord1" {
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
	if !strings.Contains(log.RequestSummary, `"sale_id":"ord1"`) {
		t.Fatalf("RequestSummary = %s", log.RequestSummary)
	}
	if strings.Contains(log.RequestSummary, "chave-pix") {
		t.Fatalf("RequestSummary must not include pix key secret: %s", log.RequestSummary)
	}
}

func TestSalesRefund_APIErrorStillWritesAudit(t *testing.T) {
	fake := &fakeSalesAPI{
		refundFn: func(ctx context.Context, id string, pixKey string) error {
			return &kiwify.APIError{
				Status:      400,
				Message:     "cannot refund",
				UserMessage: "Não é possível reembolsar.",
				Body:        `{"error":"cannot refund"}`,
			}
		},
	}
	h, s := newSalesHandler(t, fake)

	req := inertiaRequest(http.MethodPost, "/sales/ord2/refund", strings.NewReader(""))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = session.WithUserID(req, 7)
	rr := httptest.NewRecorder()
	h.Refund(rr, req, "ord2")

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303", rr.Code)
	}
	if loc := rr.Header().Get("Location"); loc != "/sales/ord2" {
		t.Fatalf("Location = %q", loc)
	}

	logs, err := s.ListAuditLogs(10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(logs) != 1 {
		t.Fatalf("audit logs = %d, want 1 even on error", len(logs))
	}
	if logs[0].ResponseStatus != 400 {
		t.Fatalf("ResponseStatus = %d, want 400", logs[0].ResponseStatus)
	}
	if !strings.Contains(logs[0].ResponseBody, "cannot refund") {
		t.Fatalf("ResponseBody = %s", logs[0].ResponseBody)
	}
}

func TestSalesRefund_GenericErrorWritesAudit500(t *testing.T) {
	fake := &fakeSalesAPI{
		refundFn: func(ctx context.Context, id string, pixKey string) error {
			return errors.New("network down")
		},
	}
	h, s := newSalesHandler(t, fake)

	req := inertiaRequest(http.MethodPost, "/sales/ord3/refund", nil)
	req = session.WithUserID(req, 1)
	rr := httptest.NewRecorder()
	h.Refund(rr, req, "ord3")

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("status = %d", rr.Code)
	}
	logs, err := s.ListAuditLogs(5, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(logs) != 1 || logs[0].ResponseStatus != 500 {
		t.Fatalf("logs = %+v", logs)
	}
}

func TestSalesRefund_RequiresAuth(t *testing.T) {
	fake := &fakeSalesAPI{}
	h, _ := newSalesHandler(t, fake)

	// Same wrapping as routes.go.
	handler := middleware.RequireAuthFunc("/login", cais.StringParam("id", h.Refund))

	req := httptest.NewRequest(http.MethodPost, "/sales/ord1/refund", nil)
	// No session user id.
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303", rr.Code)
	}
	if loc := rr.Header().Get("Location"); loc != "/login" {
		t.Fatalf("Location = %q, want /login", loc)
	}
	if fake.refundN != 0 {
		t.Fatalf("RefundSale should not be called when unauthenticated")
	}
}

func TestSalesList_RequiresAuth(t *testing.T) {
	fake := &fakeSalesAPI{}
	h, _ := newSalesHandler(t, fake)
	handler := middleware.RequireAuthFunc("/login", h.List)

	req := httptest.NewRequest(http.MethodGet, "/sales", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusSeeOther || rr.Header().Get("Location") != "/login" {
		t.Fatalf("status=%d loc=%s", rr.Code, rr.Header().Get("Location"))
	}
}
