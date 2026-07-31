package store

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"
)

type KiwifySettings struct {
	AccountID                  string
	ClientID                   string
	ClientSecretCiphertext     string
	OAuthAccessTokenCiphertext string
	TokenExpiresAt             *time.Time
	WebhookReceiveToken        string
	UpdatedAt                  time.Time
}

func (s *SQLiteStore) GetKiwifySettings() (KiwifySettings, error) {
	var ks KiwifySettings
	var tokenExpires sql.NullString
	err := s.db.QueryRow(`
		SELECT account_id, client_id, client_secret_ciphertext,
		       oauth_access_token_ciphertext, token_expires_at,
		       webhook_receive_token, updated_at
		FROM kiwify_settings WHERE id = 1
	`).Scan(
		&ks.AccountID,
		&ks.ClientID,
		&ks.ClientSecretCiphertext,
		&ks.OAuthAccessTokenCiphertext,
		&tokenExpires,
		&ks.WebhookReceiveToken,
		&ks.UpdatedAt,
	)
	if err != nil {
		return KiwifySettings{}, fmt.Errorf("get kiwify settings: %w", err)
	}
	if tokenExpires.Valid && tokenExpires.String != "" {
		t, parseErr := time.ParseInLocation("2006-01-02 15:04:05", tokenExpires.String, time.UTC)
		if parseErr != nil {
			// Also try RFC3339 in case drivers return that form
			t, parseErr = time.Parse(time.RFC3339, tokenExpires.String)
			if parseErr != nil {
				return KiwifySettings{}, fmt.Errorf("parse token_expires_at: %w", parseErr)
			}
		}
		ks.TokenExpiresAt = &t
	}
	return ks, nil
}

func (s *SQLiteStore) SaveKiwifySettings(ks KiwifySettings) error {
	token := ks.WebhookReceiveToken
	if token == "" {
		// Preserve existing token if present; otherwise generate a new one.
		existing, err := s.GetKiwifySettings()
		if err == nil && existing.WebhookReceiveToken != "" {
			token = existing.WebhookReceiveToken
		} else {
			b := make([]byte, 32)
			if _, err := rand.Read(b); err != nil {
				return fmt.Errorf("generate webhook token: %w", err)
			}
			token = hex.EncodeToString(b)
		}
	}

	var tokenExpires any
	if ks.TokenExpiresAt != nil {
		tokenExpires = ks.TokenExpiresAt.UTC().Format("2006-01-02 15:04:05")
	}

	_, err := s.db.Exec(`
		UPDATE kiwify_settings SET
			account_id = ?,
			client_id = ?,
			client_secret_ciphertext = ?,
			oauth_access_token_ciphertext = ?,
			token_expires_at = ?,
			webhook_receive_token = ?,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = 1
	`,
		ks.AccountID,
		ks.ClientID,
		ks.ClientSecretCiphertext,
		ks.OAuthAccessTokenCiphertext,
		tokenExpires,
		token,
	)
	if err != nil {
		return fmt.Errorf("save kiwify settings: %w", err)
	}
	return nil
}

func (s *SQLiteStore) UpdateOAuthToken(ciphertext string, expiresAt time.Time) error {
	expires := expiresAt.UTC().Format("2006-01-02 15:04:05")
	_, err := s.db.Exec(`
		UPDATE kiwify_settings SET
			oauth_access_token_ciphertext = ?,
			token_expires_at = ?,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = 1
	`, ciphertext, expires)
	if err != nil {
		return fmt.Errorf("update oauth token: %w", err)
	}
	return nil
}

func (s *SQLiteStore) Configured() (bool, error) {
	var accountID, clientID, secret string
	err := s.db.QueryRow(`
		SELECT account_id, client_id, client_secret_ciphertext
		FROM kiwify_settings WHERE id = 1
	`).Scan(&accountID, &clientID, &secret)
	if err != nil {
		return false, fmt.Errorf("configured: %w", err)
	}
	return accountID != "" && clientID != "" && secret != "", nil
}
