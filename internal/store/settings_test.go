package store

import (
	"encoding/hex"
	"testing"
	"time"
)

func TestKiwifySettingsRoundTrip(t *testing.T) {
	s := newTestStore(t)
	err := s.SaveKiwifySettings(KiwifySettings{
		AccountID:              "acc1",
		ClientID:               "cid",
		ClientSecretCiphertext: "enc-secret",
		WebhookReceiveToken:    "tok123",
	})
	if err != nil {
		t.Fatal(err)
	}
	got, err := s.GetKiwifySettings()
	if err != nil {
		t.Fatal(err)
	}
	if got.AccountID != "acc1" || got.ClientID != "cid" || got.WebhookReceiveToken != "tok123" {
		t.Fatalf("%+v", got)
	}
	if got.ClientSecretCiphertext != "enc-secret" {
		t.Fatalf("ClientSecretCiphertext = %q, want enc-secret", got.ClientSecretCiphertext)
	}
}

func TestSaveKiwifySettingsGeneratesWebhookToken(t *testing.T) {
	s := newTestStore(t)
	err := s.SaveKiwifySettings(KiwifySettings{
		AccountID:              "acc1",
		ClientID:               "cid",
		ClientSecretCiphertext: "enc-secret",
		// WebhookReceiveToken empty → should be generated
	})
	if err != nil {
		t.Fatal(err)
	}
	got, err := s.GetKiwifySettings()
	if err != nil {
		t.Fatal(err)
	}
	if got.WebhookReceiveToken == "" {
		t.Fatal("expected non-empty WebhookReceiveToken after save")
	}
	if _, err := hex.DecodeString(got.WebhookReceiveToken); err != nil {
		t.Fatalf("WebhookReceiveToken should be hex: %v", err)
	}
	// 32 bytes → 64 hex chars
	if len(got.WebhookReceiveToken) != 64 {
		t.Fatalf("WebhookReceiveToken len = %d, want 64", len(got.WebhookReceiveToken))
	}
}

func TestConfigured(t *testing.T) {
	s := newTestStore(t)

	ok, err := s.Configured()
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("empty settings should not be configured")
	}

	if err := s.SaveKiwifySettings(KiwifySettings{
		AccountID:              "acc1",
		ClientID:               "cid",
		ClientSecretCiphertext: "enc-secret",
	}); err != nil {
		t.Fatal(err)
	}

	ok, err = s.Configured()
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("settings with account_id, client_id, secret should be configured")
	}
}

func TestUpdateOAuthToken(t *testing.T) {
	s := newTestStore(t)
	if err := s.SaveKiwifySettings(KiwifySettings{
		AccountID:              "acc1",
		ClientID:               "cid",
		ClientSecretCiphertext: "enc-secret",
	}); err != nil {
		t.Fatal(err)
	}

	expires := time.Now().UTC().Add(time.Hour).Truncate(time.Second)
	if err := s.UpdateOAuthToken("tok-cipher", expires); err != nil {
		t.Fatal(err)
	}

	got, err := s.GetKiwifySettings()
	if err != nil {
		t.Fatal(err)
	}
	if got.OAuthAccessTokenCiphertext != "tok-cipher" {
		t.Fatalf("OAuthAccessTokenCiphertext = %q", got.OAuthAccessTokenCiphertext)
	}
	if got.TokenExpiresAt == nil {
		t.Fatal("TokenExpiresAt is nil")
	}
	// SQLite datetime may lose sub-second precision
	if got.TokenExpiresAt.Unix() != expires.Unix() {
		t.Fatalf("TokenExpiresAt = %v, want %v", got.TokenExpiresAt, expires)
	}
}

func TestInsertAuditAndList(t *testing.T) {
	s := newTestStore(t)
	id, err := s.InsertAuditLog(AuditLog{
		UserID: 1, Action: "sales.refund", ResourceType: "sale", ResourceID: "ord1",
		RequestSummary: `{"id":"ord1"}`, ResponseStatus: 200, ResponseBody: `{"refunded":true}`, IP: "127.0.0.1",
	})
	if err != nil || id == 0 {
		t.Fatalf("id=%d err=%v", id, err)
	}
	logs, err := s.ListAuditLogs(50, 0)
	if err != nil || len(logs) != 1 {
		t.Fatalf("len=%d err=%v", len(logs), err)
	}
	if logs[0].Action != "sales.refund" || logs[0].ResourceID != "ord1" {
		t.Fatalf("%+v", logs[0])
	}
}

func TestInsertWebhookEvent(t *testing.T) {
	s := newTestStore(t)
	id, err := s.InsertWebhookEvent(WebhookEvent{
		EventType: "compra_aprovada", PayloadJSON: `{"a":1}`, HeadersJSON: `{}`, ProcessedOK: true,
	})
	if err != nil || id == 0 {
		t.Fatal(err)
	}

	events, err := s.ListWebhookEvents(50, 0)
	if err != nil || len(events) != 1 {
		t.Fatalf("len=%d err=%v", len(events), err)
	}
	if events[0].EventType != "compra_aprovada" || !events[0].ProcessedOK {
		t.Fatalf("%+v", events[0])
	}
}
