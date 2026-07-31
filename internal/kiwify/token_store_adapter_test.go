package kiwify_test

import (
	"testing"
	"time"

	"github.com/puppe1990/kiwify_dashboard/internal/crypto"
	"github.com/puppe1990/kiwify_dashboard/internal/kiwify"
	"github.com/puppe1990/kiwify_dashboard/internal/store"
)

func testKey() []byte {
	return crypto.DeriveKey("test-app-secret-for-unit-tests-32b")
}

func setupSettingsStore(t *testing.T) store.Store {
	t.Helper()
	s, err := store.NewSQLiteStore(":memory:", "test")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })

	secretCT, err := crypto.Encrypt(testKey(), "client-secret-value")
	if err != nil {
		t.Fatal(err)
	}
	if err := s.SaveKiwifySettings(store.KiwifySettings{
		AccountID:              "acc-1",
		ClientID:               "cid-1",
		ClientSecretCiphertext: secretCT,
	}); err != nil {
		t.Fatal(err)
	}
	return s
}

func TestDBTokenStore_EmptyToken(t *testing.T) {
	st := setupSettingsStore(t)
	ts := &kiwify.DBTokenStore{Store: st, Key: testKey()}

	token, exp, err := ts.LoadToken()
	if err != nil {
		t.Fatal(err)
	}
	if token != "" || !exp.IsZero() {
		t.Fatalf("got token=%q exp=%v, want empty", token, exp)
	}
}

func TestDBTokenStore_RoundTrip(t *testing.T) {
	st := setupSettingsStore(t)
	ts := &kiwify.DBTokenStore{Store: st, Key: testKey()}

	expires := time.Now().UTC().Add(time.Hour).Truncate(time.Second)
	if err := ts.SaveToken("access-token-xyz", expires); err != nil {
		t.Fatal(err)
	}

	// Ciphertext in DB must not be plaintext.
	settings, err := st.GetKiwifySettings()
	if err != nil {
		t.Fatal(err)
	}
	if settings.OAuthAccessTokenCiphertext == "" || settings.OAuthAccessTokenCiphertext == "access-token-xyz" {
		t.Fatalf("expected encrypted ciphertext, got %q", settings.OAuthAccessTokenCiphertext)
	}

	token, exp, err := ts.LoadToken()
	if err != nil {
		t.Fatal(err)
	}
	if token != "access-token-xyz" {
		t.Fatalf("token = %q", token)
	}
	if exp.Unix() != expires.Unix() {
		t.Fatalf("exp = %v, want %v", exp, expires)
	}
}

func TestDBTokenStore_ClearToken(t *testing.T) {
	st := setupSettingsStore(t)
	ts := &kiwify.DBTokenStore{Store: st, Key: testKey()}

	if err := ts.SaveToken("tok", time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := ts.SaveToken("", time.Time{}); err != nil {
		t.Fatal(err)
	}
	token, exp, err := ts.LoadToken()
	if err != nil {
		t.Fatal(err)
	}
	if token != "" || !exp.IsZero() {
		t.Fatalf("after clear: token=%q exp=%v", token, exp)
	}
}

func TestNewClientFromStore(t *testing.T) {
	st := setupSettingsStore(t)
	c, err := kiwify.NewClientFromStore(st, testKey(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if c == nil {
		t.Fatal("expected client")
	}
}

func TestNewClientFromStore_Incomplete(t *testing.T) {
	st, err := store.NewSQLiteStore(":memory:", "test")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })

	_, err = kiwify.NewClientFromStore(st, testKey(), nil)
	if err == nil {
		t.Fatal("expected error for incomplete settings")
	}
}
