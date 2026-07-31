package kiwify

import (
	"fmt"
	"net/http"

	"github.com/puppe1990/kiwify_dashboard/internal/crypto"
	"github.com/puppe1990/kiwify_dashboard/internal/store"
)

// NewClientFromStore loads Kiwify credentials from the store, decrypts the
// client secret, and returns a Client with a DB-backed TokenStore.
// Returns an error when account_id, client_id, or client_secret are missing.
func NewClientFromStore(st store.Store, key []byte, httpClient *http.Client) (*Client, error) {
	if st == nil {
		return nil, fmt.Errorf("kiwify: store is required")
	}
	if len(key) == 0 {
		return nil, fmt.Errorf("kiwify: encryption key is required")
	}

	settings, err := st.GetKiwifySettings()
	if err != nil {
		return nil, err
	}
	if settings.AccountID == "" || settings.ClientID == "" || settings.ClientSecretCiphertext == "" {
		return nil, fmt.Errorf("kiwify: incomplete settings (account_id, client_id, and client_secret are required)")
	}

	secret, err := crypto.Decrypt(key, settings.ClientSecretCiphertext)
	if err != nil {
		return nil, fmt.Errorf("kiwify: decrypt client secret: %w", err)
	}

	return NewClient(Config{
		AccountID:    settings.AccountID,
		ClientID:     settings.ClientID,
		ClientSecret: secret,
		TokenStore:   &DBTokenStore{Store: st, Key: key},
		HTTPClient:   httpClient,
	}), nil
}
