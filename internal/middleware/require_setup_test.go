package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/puppe1990/kiwify_dashboard/internal/store"
)

func newTestStore(t *testing.T) store.Store {
	t.Helper()
	s, err := store.NewSQLiteStore(":memory:", "test")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func TestRequireSetup_UnconfiguredRedirects(t *testing.T) {
	st := newTestStore(t)
	nextCalled := false
	h := RequireSetup(st)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusSeeOther)
	}
	if loc := rec.Header().Get("Location"); loc != "/setup" {
		t.Fatalf("Location = %q, want /setup", loc)
	}
	if nextCalled {
		t.Fatal("next handler must not run when unconfigured")
	}
}

func TestRequireSetup_ConfiguredAllowsNext(t *testing.T) {
	st := newTestStore(t)
	if err := st.SaveKiwifySettings(store.KiwifySettings{
		AccountID:              "acc",
		ClientID:               "cid",
		ClientSecretCiphertext: "enc",
	}); err != nil {
		t.Fatal(err)
	}

	nextCalled := false
	h := RequireSetup(st)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !nextCalled {
		t.Fatal("next handler should run when configured")
	}
}

func TestRequireSetup_SkipsAuthAndSetupPaths(t *testing.T) {
	st := newTestStore(t) // unconfigured
	paths := []string{
		"/login",
		"/signup",
		"/forgot-password",
		"/reset-password",
		"/setup",
		"/webhooks/kiwify/abc",
		"/static/css/styles.css",
		"/health",
	}
	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			nextCalled := false
			h := RequireSetup(st)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				nextCalled = true
				w.WriteHeader(http.StatusOK)
			}))
			req := httptest.NewRequest(http.MethodGet, path, nil)
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			if rec.Code != http.StatusOK || !nextCalled {
				t.Fatalf("path %s: status=%d next=%v (should skip setup check)", path, rec.Code, nextCalled)
			}
		})
	}
}
