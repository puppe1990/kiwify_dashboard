package handlers

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/puppe1990/cais/pkg/cais"
	"github.com/puppe1990/kiwify_dashboard/internal/crypto"
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
}
