package kiwify_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/puppe1990/kiwify_dashboard/internal/kiwify"
)

type memTokenStore struct {
	token string
	exp   time.Time
}

func (m *memTokenStore) LoadToken() (string, time.Time, error) { return m.token, m.exp, nil }
func (m *memTokenStore) SaveToken(token string, exp time.Time) error {
	m.token, m.exp = token, exp
	return nil
}

func TestGetTokenCachesUntilExpiry(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/oauth/token" {
			calls.Add(1)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": "tok-abc",
				"token_type":   "Bearer",
				"expires_in":   "3600",
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	ts := &memTokenStore{}
	c := kiwify.NewClient(kiwify.Config{
		BaseURL:      srv.URL + "/v1",
		AccountID:    "acc",
		ClientID:     "cid",
		ClientSecret: "sec",
		TokenStore:   ts,
		HTTPClient:   srv.Client(),
	})
	tok1, err := c.GetToken(context.Background())
	if err != nil || tok1 != "tok-abc" {
		t.Fatalf("%v %q", err, tok1)
	}
	tok2, err := c.GetToken(context.Background())
	if err != nil || tok2 != "tok-abc" {
		t.Fatal(err)
	}
	if calls.Load() != 1 {
		t.Fatalf("oauth called %d times", calls.Load())
	}
}

func TestGetTokenReusesPreloadedValidToken(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/oauth/token" {
			calls.Add(1)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": "fresh",
				"expires_in":   3600,
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	ts := &memTokenStore{
		token: "preloaded",
		exp:   time.Now().Add(2 * time.Hour),
	}
	c := kiwify.NewClient(kiwify.Config{
		BaseURL:      srv.URL + "/v1",
		AccountID:    "acc",
		ClientID:     "cid",
		ClientSecret: "sec",
		TokenStore:   ts,
		HTTPClient:   srv.Client(),
	})
	tok, err := c.GetToken(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if tok != "preloaded" {
		t.Fatalf("got %q, want preloaded", tok)
	}
	if calls.Load() != 0 {
		t.Fatalf("oauth called %d times, want 0", calls.Load())
	}
}

func TestGetTokenExpiresInAsNumber(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/oauth/token" {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": "num-tok",
				"expires_in":   7200,
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	ts := &memTokenStore{}
	c := kiwify.NewClient(kiwify.Config{
		BaseURL:      srv.URL + "/v1",
		AccountID:    "acc",
		ClientID:     "cid",
		ClientSecret: "sec",
		TokenStore:   ts,
		HTTPClient:   srv.Client(),
	})
	tok, err := c.GetToken(context.Background())
	if err != nil || tok != "num-tok" {
		t.Fatalf("%v %q", err, tok)
	}
	if ts.token != "num-tok" {
		t.Fatalf("store token = %q", ts.token)
	}
	// expires_in 7200s → should be well over an hour from now
	if ts.exp.Before(time.Now().Add(90 * time.Minute)) {
		t.Fatalf("expires too soon: %v", ts.exp)
	}
}

func TestDoSendsAuthHeaders(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/oauth/token" {
			_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "t", "expires_in": "3600"})
			return
		}
		if r.Header.Get("Authorization") != "Bearer t" {
			t.Errorf("auth %q", r.Header.Get("Authorization"))
		}
		if r.Header.Get("x-kiwify-account-id") != "acc" {
			t.Errorf("account %q", r.Header.Get("x-kiwify-account-id"))
		}
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	c := kiwify.NewClient(kiwify.Config{
		BaseURL: srv.URL + "/v1", AccountID: "acc", ClientID: "c", ClientSecret: "s",
		TokenStore: &memTokenStore{}, HTTPClient: srv.Client(),
	})
	var out map[string]any
	if err := c.GetJSON(context.Background(), "/products", nil, &out); err != nil {
		t.Fatal(err)
	}
}

func TestAPIError429(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/oauth/token" {
			_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "t", "expires_in": "3600"})
			return
		}
		w.WriteHeader(429)
		_, _ = w.Write([]byte(`{"message":"rate limited"}`))
	}))
	defer srv.Close()
	c := kiwify.NewClient(kiwify.Config{
		BaseURL: srv.URL + "/v1", AccountID: "a", ClientID: "c", ClientSecret: "s",
		TokenStore: &memTokenStore{}, HTTPClient: srv.Client(),
	})
	err := c.GetJSON(context.Background(), "/sales", map[string]string{"start_date": "x", "end_date": "y"}, nil)
	apiErr, ok := kiwify.AsAPIError(err)
	if !ok || apiErr.Status != 429 {
		t.Fatalf("%v", err)
	}
	if apiErr.UserMessage == "" {
		t.Fatal("expected PT-BR user message")
	}
}
