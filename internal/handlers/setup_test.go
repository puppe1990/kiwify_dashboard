package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/puppe1990/cais/pkg/cais"
	"github.com/puppe1990/kiwify_dashboard/internal/crypto"
	"github.com/puppe1990/kiwify_dashboard/internal/store"
)

func testAppSecret() []byte {
	return crypto.DeriveKey("test-app-secret-for-unit-tests-32b")
}

func TestSetup_Get_rendersSetup(t *testing.T) {
	s := setupTestStore(t)
	h := NewSetupHandler(s, testAppSecret(), testSite(), cais.Config{AppURL: "https://cais.example.com"}, setupTestInertia(t))

	req := inertiaRequest(http.MethodGet, "/setup", nil)
	rr := httptest.NewRecorder()
	h.Get(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rr.Code)
	}
	assertInertiaComponent(t, rr, "Setup")
}

func TestSetup_Post_validationErrors(t *testing.T) {
	s := setupTestStore(t)
	h := NewSetupHandler(s, testAppSecret(), testSite(), cais.Config{AppURL: "https://cais.example.com"}, setupTestInertia(t))

	form := url.Values{}
	req := inertiaRequest(http.MethodPost, "/setup", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	h.Post(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rr.Code)
	}
	assertInertiaComponent(t, rr, "Setup")
	assertInertiaErrors(t, rr, "client_id", "client_secret", "account_id")
}

func TestSetup_Post_savesEncryptedCredentials(t *testing.T) {
	s := setupTestStore(t)
	h := NewSetupHandler(s, testAppSecret(), testSite(), cais.Config{AppURL: "https://cais.example.com"}, setupTestInertia(t))

	// Avoid real network OAuth probe in unit tests.
	oldProbe := probeKiwifyOAuth
	probeKiwifyOAuth = func(ctx context.Context, st store.Store, key []byte) error { return nil }
	t.Cleanup(func() { probeKiwifyOAuth = oldProbe })

	plainSecret := "my-super-secret-value"
	form := url.Values{
		"client_id":     {"cid-123"},
		"client_secret": {plainSecret},
		"account_id":    {"acc-456"},
	}
	req := httptest.NewRequest(http.MethodPost, "/setup", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	h.Post(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303, body: %s", rr.Code, rr.Body.String())
	}
	if loc := rr.Header().Get("Location"); loc != "/dashboard" {
		t.Errorf("Location = %q, want /dashboard", loc)
	}

	ok, err := s.Configured()
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected Configured true after setup")
	}

	settings, err := s.GetKiwifySettings()
	if err != nil {
		t.Fatal(err)
	}
	if settings.AccountID != "acc-456" || settings.ClientID != "cid-123" {
		t.Fatalf("settings = %+v", settings)
	}
	if settings.ClientSecretCiphertext == "" {
		t.Fatal("ClientSecretCiphertext is empty")
	}
	if settings.ClientSecretCiphertext == plainSecret {
		t.Fatal("ClientSecretCiphertext must not equal plaintext")
	}
	if settings.WebhookReceiveToken == "" {
		t.Fatal("WebhookReceiveToken should be generated")
	}

	// Ciphertext must decrypt to original secret
	pt, err := crypto.Decrypt(testAppSecret(), settings.ClientSecretCiphertext)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if pt != plainSecret {
		t.Fatalf("decrypted = %q, want %q", pt, plainSecret)
	}

	// Flash cookie must be set so dashboard can show success/error feedback.
	if c := rr.Result().Cookies(); len(c) == 0 {
		// At least one Set-Cookie expected (cais_flash)
	}
	foundFlash := false
	for _, c := range rr.Result().Cookies() {
		if c.Name == "cais_flash" && c.Value != "" {
			foundFlash = true
		}
	}
	if !foundFlash {
		t.Fatal("expected cais_flash cookie after setup (user-facing feedback)")
	}
}

func TestSetup_Post_oauthFailureStillSavesAndFlashesError(t *testing.T) {
	s := setupTestStore(t)
	h := NewSetupHandler(s, testAppSecret(), testSite(), cais.Config{AppURL: "https://cais.example.com"}, setupTestInertia(t))

	oldProbe := probeKiwifyOAuth
	probeKiwifyOAuth = func(ctx context.Context, st store.Store, key []byte) error {
		return &kiwifyAPIError{msg: "invalid client"}
	}
	t.Cleanup(func() { probeKiwifyOAuth = oldProbe })

	form := url.Values{
		"client_id":     {"cid"},
		"client_secret": {"sec"},
		"account_id":    {"acc"},
	}
	req := httptest.NewRequest(http.MethodPost, "/setup", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	h.Post(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303", rr.Code)
	}
	ok, err := s.Configured()
	if err != nil || !ok {
		t.Fatalf("configured=%v err=%v", ok, err)
	}
	foundFlash := false
	for _, c := range rr.Result().Cookies() {
		if c.Name == "cais_flash" && c.Value != "" {
			foundFlash = true
		}
	}
	if !foundFlash {
		t.Fatal("expected error flash cookie when OAuth fails")
	}
}

// kiwifyAPIError is a tiny error used only to exercise userFacingAPIError path when needed.
type kiwifyAPIError struct{ msg string }

func (e *kiwifyAPIError) Error() string { return e.msg }
