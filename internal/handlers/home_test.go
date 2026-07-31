package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/puppe1990/cais/pkg/cais"
	"github.com/puppe1990/cais/pkg/cais/i18n"
	"github.com/puppe1990/cais/pkg/cais/session"

	"github.com/puppe1990/kiwify_dashboard/internal/crypto"
	"github.com/puppe1990/kiwify_dashboard/internal/store"
)

func newHomeHandler(t *testing.T) *HomeHandler {
	t.Helper()
	return NewHomeHandler(setupTestRenderer(t), testSite(), i18n.DefaultCatalog(), cais.Config{}, setupTestInertia(t))
}

func TestHomeHandler_Returns200(t *testing.T) {
	h := newHomeHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusOK)
	}
}

func TestHomeHandler_InertiaComponent(t *testing.T) {
	h := newHomeHandler(t)

	req := inertiaRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	assertInertiaComponent(t, rr, "Home")
}

func TestHomeHandler_InertiaShell(t *testing.T) {
	h := newHomeHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	body := rr.Body.String()
	if !strings.Contains(body, `id="app"`) && !strings.Contains(body, "data-page") {
		t.Errorf("body missing Inertia shell markers, got: %s", body)
	}
}

func TestHomeHandler_ContentType(t *testing.T) {
	h := newHomeHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	ct := rr.Header().Get("Content-Type")
	if !strings.Contains(ct, "text/html") {
		t.Errorf("Content-Type = %q, want text/html", ct)
	}
}

func TestHomeHandler_AuthenticatedConfigured_RedirectsDashboard(t *testing.T) {
	s := setupTestStore(t)
	// Mark settings as configured.
	if err := s.SaveKiwifySettings(store.KiwifySettings{
		AccountID:              "acc",
		ClientID:               "cid",
		ClientSecretCiphertext: mustEncrypt(t, "secret"),
		WebhookReceiveToken:    "tok",
	}); err != nil {
		t.Fatal(err)
	}

	h := NewHomeHandlerWithStore(setupTestRenderer(t), s, testSite(), i18n.DefaultCatalog(), cais.Config{}, setupTestInertia(t))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = session.WithUserID(req, 1)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303", rr.Code)
	}
	if loc := rr.Header().Get("Location"); loc != "/dashboard" {
		t.Fatalf("Location = %q, want /dashboard", loc)
	}
}

func TestHomeHandler_AuthenticatedUnconfigured_RedirectsSetup(t *testing.T) {
	s := setupTestStore(t)
	h := NewHomeHandlerWithStore(setupTestRenderer(t), s, testSite(), i18n.DefaultCatalog(), cais.Config{}, setupTestInertia(t))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = session.WithUserID(req, 1)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303", rr.Code)
	}
	if loc := rr.Header().Get("Location"); loc != "/setup" {
		t.Fatalf("Location = %q, want /setup", loc)
	}
}

func mustEncrypt(t *testing.T, plain string) string {
	t.Helper()
	ct, err := crypto.Encrypt(testAppSecret(), plain)
	if err != nil {
		t.Fatal(err)
	}
	return ct
}
