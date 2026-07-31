package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/puppe1990/cais/pkg/cais"
	"github.com/puppe1990/cais/pkg/cais/middleware"
	"github.com/puppe1990/kiwify_dashboard/internal/kiwify"
	"github.com/puppe1990/kiwify_dashboard/internal/store"
)

type fakeAccountAPI struct {
	getFn func(ctx context.Context) (kiwify.Account, error)
	getN  int
}

func (f *fakeAccountAPI) GetAccount(ctx context.Context) (kiwify.Account, error) {
	f.getN++
	if f.getFn != nil {
		return f.getFn(ctx)
	}
	return kiwify.Account{
		ID:    "acc-1",
		Name:  "Conta Demo",
		Email: "demo@kiwify.com",
		Raw: map[string]any{
			"id":    "acc-1",
			"name":  "Conta Demo",
			"email": "demo@kiwify.com",
		},
	}, nil
}

func newAccountHandler(t *testing.T, api AccountAPI) (*AccountHandler, store.Store) {
	t.Helper()
	s := setupTestStore(t)
	h := NewAccountHandler(s, testAppSecret(), testSite(), cais.Config{}, setupTestInertia(t), api)
	return h, s
}

func TestAccountGet_RendersAccount(t *testing.T) {
	fake := &fakeAccountAPI{
		getFn: func(ctx context.Context) (kiwify.Account, error) {
			return kiwify.Account{
				ID:    "acc-42",
				Name:  "Minha Conta",
				Email: "owner@example.com",
				Raw: map[string]any{
					"id":    "acc-42",
					"name":  "Minha Conta",
					"email": "owner@example.com",
					"plan":  "pro",
				},
			}, nil
		},
	}
	h, _ := newAccountHandler(t, fake)

	req := inertiaRequest(http.MethodGet, "/account", nil)
	rr := httptest.NewRecorder()
	h.Get(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rr.Code, rr.Body.String())
	}
	assertInertiaComponent(t, rr, "Account")
	acc := assertInertiaProp(t, rr, "account").(map[string]any)
	if acc["id"] != "acc-42" || acc["name"] != "Minha Conta" || acc["email"] != "owner@example.com" {
		t.Fatalf("account = %v", acc)
	}
	raw, ok := acc["raw"].(map[string]any)
	if !ok || raw["plan"] != "pro" {
		t.Fatalf("raw = %v", acc["raw"])
	}
	if fake.getN != 1 {
		t.Fatalf("GetAccount called %d times", fake.getN)
	}
}

func TestAccountGet_APIError(t *testing.T) {
	fake := &fakeAccountAPI{
		getFn: func(ctx context.Context) (kiwify.Account, error) {
			return kiwify.Account{}, &kiwify.APIError{
				Status:      500,
				Message:     "boom",
				UserMessage: "Erro no servidor da Kiwify. Tente mais tarde.",
			}
		},
	}
	h, _ := newAccountHandler(t, fake)

	req := inertiaRequest(http.MethodGet, "/account", nil)
	rr := httptest.NewRecorder()
	h.Get(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d", rr.Code)
	}
	assertInertiaComponent(t, rr, "Account")
	assertInertiaErrors(t, rr, "account")
}

func TestAccountGet_AuthRedirect(t *testing.T) {
	fake := &fakeAccountAPI{}
	h, _ := newAccountHandler(t, fake)

	handler := middleware.RequireAuthFunc("/login", h.Get)
	req := httptest.NewRequest(http.MethodGet, "/account", nil)
	rr := httptest.NewRecorder()
	handler(rr, req)

	if rr.Code != http.StatusSeeOther || rr.Header().Get("Location") != "/login" {
		t.Fatalf("status=%d loc=%s", rr.Code, rr.Header().Get("Location"))
	}
	if fake.getN != 0 {
		t.Fatalf("GetAccount should not be called when unauthenticated")
	}
}
