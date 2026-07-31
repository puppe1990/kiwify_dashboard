package kiwify

import (
	"fmt"
	"time"

	"github.com/puppe1990/kiwify_dashboard/internal/crypto"
	"github.com/puppe1990/kiwify_dashboard/internal/store"
)

// DBTokenStore implements TokenStore by encrypting tokens in the app store.
// Key must be a 32-byte AES key (e.g. crypto.DeriveKey(APP_SECRET)).
type DBTokenStore struct {
	Store store.Store
	Key   []byte
}

// LoadToken decrypts the OAuth access token from settings.
// Empty ciphertext returns ("", zero time, nil).
func (t *DBTokenStore) LoadToken() (string, time.Time, error) {
	if t == nil || t.Store == nil {
		return "", time.Time{}, fmt.Errorf("kiwify: DBTokenStore is not configured")
	}
	s, err := t.Store.GetKiwifySettings()
	if err != nil {
		return "", time.Time{}, err
	}
	if s.OAuthAccessTokenCiphertext == "" {
		return "", time.Time{}, nil
	}
	token, err := crypto.Decrypt(t.Key, s.OAuthAccessTokenCiphertext)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("kiwify: decrypt oauth token: %w", err)
	}
	var exp time.Time
	if s.TokenExpiresAt != nil {
		exp = *s.TokenExpiresAt
	}
	return token, exp, nil
}

// SaveToken encrypts and persists the OAuth access token.
// An empty token clears the stored ciphertext (used to invalidate cache).
func (t *DBTokenStore) SaveToken(token string, exp time.Time) error {
	if t == nil || t.Store == nil {
		return fmt.Errorf("kiwify: DBTokenStore is not configured")
	}
	if token == "" {
		return t.Store.UpdateOAuthToken("", exp)
	}
	ct, err := crypto.Encrypt(t.Key, token)
	if err != nil {
		return fmt.Errorf("kiwify: encrypt oauth token: %w", err)
	}
	return t.Store.UpdateOAuthToken(ct, exp)
}
