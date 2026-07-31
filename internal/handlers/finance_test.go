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

type fakeFinanceAPI struct {
	balancesFn func(ctx context.Context) (kiwify.Balances, error)
	listFn     func(ctx context.Context, q kiwify.PageQuery) (kiwify.PayoutsPage, error)
	createFn   func(ctx context.Context, amount float64) (kiwify.Payout, error)

	lastQuery  kiwify.PageQuery
	createAmt  float64
	createN    int
	balancesN  int
	listN      int
}

func (f *fakeFinanceAPI) ListBalances(ctx context.Context) (kiwify.Balances, error) {
	f.balancesN++
	if f.balancesFn != nil {
		return f.balancesFn(ctx)
	}
	return kiwify.Balances{Available: 10000, Pending: 500}, nil
}

func (f *fakeFinanceAPI) ListPayouts(ctx context.Context, q kiwify.PageQuery) (kiwify.PayoutsPage, error) {
	f.listN++
	f.lastQuery = q
	if f.listFn != nil {
		return f.listFn(ctx, q)
	}
	return kiwify.PayoutsPage{Data: []kiwify.Payout{}}, nil
}

func (f *fakeFinanceAPI) CreatePayout(ctx context.Context, amount float64) (kiwify.Payout, error) {
	f.createN++
	f.createAmt = amount
	if f.createFn != nil {
		return f.createFn(ctx, amount)
	}
	return kiwify.Payout{ID: "payout-1", Amount: amount, Status: "pending"}, nil
}

func newFinanceHandler(t *testing.T, api FinanceAPI) (*FinanceHandler, store.Store) {
	t.Helper()
	s := setupTestStore(t)
	h := NewFinanceHandler(s, testAppSecret(), testSite(), cais.Config{}, setupTestInertia(t), api)
	return h, s
}

func TestFinanceGet_RendersBalancesAndPayouts(t *testing.T) {
	fake := &fakeFinanceAPI{
		balancesFn: func(ctx context.Context) (kiwify.Balances, error) {
			return kiwify.Balances{Available: 236961, Pending: 4621, LegalEntityID: "ent-1"}, nil
		},
		listFn: func(ctx context.Context, q kiwify.PageQuery) (kiwify.PayoutsPage, error) {
			return kiwify.PayoutsPage{
				Data: []kiwify.Payout{
					{ID: "po-1", Amount: 5000, Status: "paid"},
				},
				Pagination: kiwify.Pagination{Count: 1, PageNumber: 1, PageSize: 50},
			}, nil
		},
	}
	h, _ := newFinanceHandler(t, fake)

	req := inertiaRequest(http.MethodGet, "/finance", nil)
	rr := httptest.NewRecorder()
	h.Get(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rr.Code, rr.Body.String())
	}
	assertInertiaComponent(t, rr, "Finance")
	bal := assertInertiaProp(t, rr, "balances").(map[string]any)
	if bal["available"] != float64(236961) {
		t.Fatalf("balances = %v", bal)
	}
	payouts := assertInertiaProp(t, rr, "payouts").([]any)
	if len(payouts) != 1 {
		t.Fatalf("payouts len = %d", len(payouts))
	}
	if fake.lastQuery.PageSize != "50" {
		t.Fatalf("page size = %q", fake.lastQuery.PageSize)
	}
}

func TestFinanceGet_PartialErrors(t *testing.T) {
	fake := &fakeFinanceAPI{
		balancesFn: func(ctx context.Context) (kiwify.Balances, error) {
			return kiwify.Balances{}, &kiwify.APIError{
				Status: 500, Message: "x", UserMessage: "Erro de saldo.",
			}
		},
		listFn: func(ctx context.Context, q kiwify.PageQuery) (kiwify.PayoutsPage, error) {
			return kiwify.PayoutsPage{
				Data: []kiwify.Payout{{ID: "po-1", Amount: 100}},
			}, nil
		},
	}
	h, _ := newFinanceHandler(t, fake)

	req := inertiaRequest(http.MethodGet, "/finance", nil)
	rr := httptest.NewRecorder()
	h.Get(rr, req)

	assertInertiaComponent(t, rr, "Finance")
	assertInertiaErrors(t, rr, "balances")
	payouts := assertInertiaProp(t, rr, "payouts").([]any)
	if len(payouts) != 1 {
		t.Fatalf("payouts should still load: %v", payouts)
	}
}

func TestFinanceCreatePayout_SuccessWritesAudit(t *testing.T) {
	fake := &fakeFinanceAPI{}
	h, s := newFinanceHandler(t, fake)

	form := url.Values{"amount": {"5000"}}
	req := inertiaRequest(http.MethodPost, "/finance/payouts", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = session.WithUserID(req, 42)
	req.RemoteAddr = "203.0.113.10:9999"
	rr := httptest.NewRecorder()
	h.CreatePayout(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("status = %d body=%s", rr.Code, rr.Body.String())
	}
	if loc := rr.Header().Get("Location"); loc != "/finance" {
		t.Fatalf("Location = %q", loc)
	}
	if fake.createN != 1 || fake.createAmt != 5000 {
		t.Fatalf("create: n=%d amt=%v", fake.createN, fake.createAmt)
	}

	logs, err := s.ListAuditLogs(10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(logs) != 1 {
		t.Fatalf("audit logs = %d", len(logs))
	}
	log := logs[0]
	if log.Action != "finance.payout" || log.ResourceType != "payout" {
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
	if !strings.Contains(log.RequestSummary, `"amount":5000`) {
		t.Fatalf("RequestSummary = %s", log.RequestSummary)
	}
	if log.ResourceID != "payout-1" {
		t.Fatalf("ResourceID = %s", log.ResourceID)
	}
}

func TestFinanceCreatePayout_APIErrorStillWritesAudit(t *testing.T) {
	fake := &fakeFinanceAPI{
		createFn: func(ctx context.Context, amount float64) (kiwify.Payout, error) {
			return kiwify.Payout{}, &kiwify.APIError{
				Status:      400,
				Message:     "insufficient funds",
				UserMessage: "Saldo insuficiente.",
				Body:        `{"error":"insufficient funds"}`,
			}
		},
	}
	h, s := newFinanceHandler(t, fake)

	form := url.Values{"amount": {"999999"}}
	req := inertiaRequest(http.MethodPost, "/finance/payouts", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = session.WithUserID(req, 7)
	rr := httptest.NewRecorder()
	h.CreatePayout(rr, req)

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
	if logs[0].Action != "finance.payout" {
		t.Fatalf("action = %s", logs[0].Action)
	}
	if logs[0].ResponseStatus != 400 {
		t.Fatalf("ResponseStatus = %d", logs[0].ResponseStatus)
	}
	if !strings.Contains(logs[0].ResponseBody, "insufficient funds") {
		t.Fatalf("ResponseBody = %s", logs[0].ResponseBody)
	}
}

func TestFinanceCreatePayout_GenericErrorWritesAudit500(t *testing.T) {
	fake := &fakeFinanceAPI{
		createFn: func(ctx context.Context, amount float64) (kiwify.Payout, error) {
			return kiwify.Payout{}, errors.New("network down")
		},
	}
	h, s := newFinanceHandler(t, fake)

	form := url.Values{"amount": {"100"}}
	req := inertiaRequest(http.MethodPost, "/finance/payouts", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = session.WithUserID(req, 1)
	rr := httptest.NewRecorder()
	h.CreatePayout(rr, req)

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

func TestFinanceCreatePayout_InvalidAmountNoAPICall(t *testing.T) {
	fake := &fakeFinanceAPI{}
	h, s := newFinanceHandler(t, fake)

	form := url.Values{"amount": {"-10"}}
	req := inertiaRequest(http.MethodPost, "/finance/payouts", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = session.WithUserID(req, 1)
	rr := httptest.NewRecorder()
	h.CreatePayout(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("status = %d", rr.Code)
	}
	if fake.createN != 0 {
		t.Fatalf("CreatePayout should not be called for invalid amount")
	}
	logs, err := s.ListAuditLogs(5, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(logs) != 0 {
		t.Fatalf("no audit for client validation errors, got %d", len(logs))
	}
}

func TestFinanceGet_RequiresAuth(t *testing.T) {
	fake := &fakeFinanceAPI{}
	h, _ := newFinanceHandler(t, fake)
	handler := middleware.RequireAuthFunc("/login", h.Get)

	req := httptest.NewRequest(http.MethodGet, "/finance", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusSeeOther || rr.Header().Get("Location") != "/login" {
		t.Fatalf("status=%d loc=%s", rr.Code, rr.Header().Get("Location"))
	}
	if fake.balancesN != 0 || fake.listN != 0 {
		t.Fatalf("API should not be called when unauthenticated")
	}
}

func TestFinanceCreatePayout_RequiresAuth(t *testing.T) {
	fake := &fakeFinanceAPI{}
	h, _ := newFinanceHandler(t, fake)
	handler := middleware.RequireAuthFunc("/login", h.CreatePayout)

	req := httptest.NewRequest(http.MethodPost, "/finance/payouts", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusSeeOther || rr.Header().Get("Location") != "/login" {
		t.Fatalf("status=%d loc=%s", rr.Code, rr.Header().Get("Location"))
	}
	if fake.createN != 0 {
		t.Fatalf("CreatePayout should not be called when unauthenticated")
	}
}
