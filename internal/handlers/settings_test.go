package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/puppe1990/cais/pkg/cais"
	"github.com/puppe1990/kiwify_dashboard/internal/crypto"
	"github.com/puppe1990/kiwify_dashboard/internal/store"
)

func seedSettings(t *testing.T, s store.Store, secret string) store.KiwifySettings {
	t.Helper()
	ct, err := crypto.Encrypt(testAppSecret(), secret)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.SaveKiwifySettings(store.KiwifySettings{
		AccountID:              "acc-old",
		ClientID:               "cid-old",
		ClientSecretCiphertext: ct,
		WebhookReceiveToken:    "fixed-webhook-token-hex",
	}); err != nil {
		t.Fatal(err)
	}
	got, err := s.GetKiwifySettings()
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func TestSettings_Get_propsNeverIncludePlaintextSecret(t *testing.T) {
	s := setupTestStore(t)
	plain := "never-expose-this-secret"
	seedSettings(t, s, plain)

	h := NewSettingsHandler(s, testAppSecret(), testSite(), cais.Config{AppURL: "https://cais.example.com"}, setupTestInertia(t))

	req := inertiaRequest(http.MethodGet, "/settings", nil)
	rr := httptest.NewRecorder()
	h.Get(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 body=%s", rr.Code, rr.Body.String())
	}
	assertInertiaComponent(t, rr, "Settings")

	// hasSecret true, account/client present, webhook URL absolute
	if v := assertInertiaProp(t, rr, "hasSecret"); v != true {
		t.Errorf("hasSecret = %v, want true", v)
	}
	if v := assertInertiaProp(t, rr, "accountId"); v != "acc-old" {
		t.Errorf("accountId = %v", v)
	}
	if v := assertInertiaProp(t, rr, "clientId"); v != "cid-old" {
		t.Errorf("clientId = %v", v)
	}
	wantURL := "https://cais.example.com/webhooks/kiwify/fixed-webhook-token-hex"
	if v := assertInertiaProp(t, rr, "webhookReceiveURL"); v != wantURL {
		t.Errorf("webhookReceiveURL = %v, want %s", v, wantURL)
	}

	// Full JSON body must not contain plaintext secret
	body := rr.Body.String()
	if strings.Contains(body, plain) {
		t.Fatal("response body contains plaintext client secret")
	}
	// Also ensure no clientSecret / client_secret keys with the secret
	var payload map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	props, _ := payload["props"].(map[string]any)
	for _, key := range []string{"clientSecret", "client_secret", "clientSecretCiphertext", "secret"} {
		if _, ok := props[key]; ok {
			t.Errorf("props must not include %q", key)
		}
	}
}

func stubOAuthOK(t *testing.T) {
	t.Helper()
	old := probeKiwifyOAuth
	probeKiwifyOAuth = func(ctx context.Context, st store.Store, key []byte) error { return nil }
	t.Cleanup(func() { probeKiwifyOAuth = old })
}

func TestSettings_Get_includesApiStatus(t *testing.T) {
	s := setupTestStore(t)
	seedSettings(t, s, "secret")
	h := NewSettingsHandler(s, testAppSecret(), testSite(), cais.Config{AppURL: "https://cais.example.com"}, setupTestInertia(t))

	req := inertiaRequest(http.MethodGet, "/settings", nil)
	rr := httptest.NewRecorder()
	h.Get(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d", rr.Code)
	}
	if v := assertInertiaProp(t, rr, "apiStatus"); v != "untested" {
		t.Errorf("apiStatus = %v, want untested (no oauth token yet)", v)
	}
	if v := assertInertiaProp(t, rr, "apiStatusMessage"); v == nil || v == "" {
		t.Error("apiStatusMessage should be set")
	}
}

func TestSettings_Post_emptySecretKeepsPreviousCiphertext(t *testing.T) {
	s := setupTestStore(t)
	seeded := seedSettings(t, s, "original-secret")
	prevCipher := seeded.ClientSecretCiphertext
	prevToken := seeded.WebhookReceiveToken

	h := NewSettingsHandler(s, testAppSecret(), testSite(), cais.Config{AppURL: "https://cais.example.com"}, setupTestInertia(t))
	stubOAuthOK(t)

	form := url.Values{
		"client_id":     {"cid-new"},
		"account_id":    {"acc-new"},
		"client_secret": {""}, // keep previous
	}
	req := httptest.NewRequest(http.MethodPost, "/settings", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	h.Post(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303 body=%s", rr.Code, rr.Body.String())
	}
	if loc := rr.Header().Get("Location"); loc != "/settings" {
		t.Errorf("Location = %q, want /settings", loc)
	}

	got, err := s.GetKiwifySettings()
	if err != nil {
		t.Fatal(err)
	}
	if got.AccountID != "acc-new" || got.ClientID != "cid-new" {
		t.Fatalf("ids = %s/%s", got.AccountID, got.ClientID)
	}
	if got.ClientSecretCiphertext != prevCipher {
		t.Fatal("ClientSecretCiphertext changed when secret was empty")
	}
	if got.WebhookReceiveToken != prevToken {
		t.Fatalf("WebhookReceiveToken = %q, want %q", got.WebhookReceiveToken, prevToken)
	}
}

func TestSettings_Post_updatesSecretWhenProvided(t *testing.T) {
	s := setupTestStore(t)
	seeded := seedSettings(t, s, "original-secret")
	prevCipher := seeded.ClientSecretCiphertext

	h := NewSettingsHandler(s, testAppSecret(), testSite(), cais.Config{AppURL: "https://cais.example.com"}, setupTestInertia(t))
	stubOAuthOK(t)

	newSecret := "brand-new-secret"
	form := url.Values{
		"client_id":     {"cid-new"},
		"account_id":    {"acc-new"},
		"client_secret": {newSecret},
	}
	req := httptest.NewRequest(http.MethodPost, "/settings", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	h.Post(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303", rr.Code)
	}

	got, err := s.GetKiwifySettings()
	if err != nil {
		t.Fatal(err)
	}
	if got.ClientSecretCiphertext == prevCipher {
		t.Fatal("ClientSecretCiphertext should change when new secret provided")
	}
	if got.ClientSecretCiphertext == newSecret {
		t.Fatal("ciphertext must not equal plaintext")
	}
	pt, err := crypto.Decrypt(testAppSecret(), got.ClientSecretCiphertext)
	if err != nil {
		t.Fatal(err)
	}
	if pt != newSecret {
		t.Fatalf("decrypted = %q, want %q", pt, newSecret)
	}

	foundFlash := false
	for _, c := range rr.Result().Cookies() {
		if c.Name == "cais_flash" && c.Value != "" {
			foundFlash = true
		}
	}
	if !foundFlash {
		t.Fatal("expected cais_flash cookie after settings save")
	}
}

func TestSettings_Post_validationErrors(t *testing.T) {
	s := setupTestStore(t)
	seedSettings(t, s, "secret")
	h := NewSettingsHandler(s, testAppSecret(), testSite(), cais.Config{AppURL: "https://cais.example.com"}, setupTestInertia(t))

	form := url.Values{"client_id": {""}, "account_id": {""}}
	req := inertiaRequest(http.MethodPost, "/settings", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	h.Post(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rr.Code)
	}
	assertInertiaComponent(t, rr, "Settings")
	assertInertiaErrors(t, rr, "client_id", "account_id")
}
