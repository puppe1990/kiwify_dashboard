package kiwify

import (
	"context"
	"encoding/json"
)

// Account is a flexible view of GET /account.
// Known fields are decoded when present; Raw keeps the full API payload for UI display.
type Account struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	// Raw is the full decoded JSON object (or wrapper data) for unknown fields.
	Raw map[string]any `json:"-"`
}

// GetAccount fetches account details from GET /account.
// Response shapes vary; we accept a flat object or a { "data": {...} } wrapper.
func (c *Client) GetAccount(ctx context.Context) (Account, error) {
	var raw json.RawMessage
	if err := c.GetJSON(ctx, "/account", nil, &raw); err != nil {
		return Account{}, err
	}
	return decodeAccount(raw)
}

func decodeAccount(raw json.RawMessage) (Account, error) {
	if len(raw) == 0 {
		return Account{Raw: map[string]any{}}, nil
	}

	// Prefer unwrapping { "data": {...} } when present.
	var wrapper struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(raw, &wrapper); err == nil && len(wrapper.Data) > 0 && wrapper.Data[0] == '{' {
		raw = wrapper.Data
	}

	var acc Account
	_ = json.Unmarshal(raw, &acc)

	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		// Non-object payload — still return something usable.
		return Account{
			Raw: map[string]any{"value": json.RawMessage(raw)},
		}, nil
	}
	if m == nil {
		m = map[string]any{}
	}
	acc.Raw = m

	// Fallback field names sometimes used by APIs.
	if acc.ID == "" {
		if v, ok := stringFromMap(m, "account_id", "accountId"); ok {
			acc.ID = v
		}
	}
	if acc.Name == "" {
		if v, ok := stringFromMap(m, "company_name", "companyName", "full_name", "fullName"); ok {
			acc.Name = v
		}
	}
	if acc.Email == "" {
		if v, ok := stringFromMap(m, "owner_email", "ownerEmail"); ok {
			acc.Email = v
		}
	}
	return acc, nil
}

func stringFromMap(m map[string]any, keys ...string) (string, bool) {
	for _, k := range keys {
		if v, ok := m[k]; ok {
			if s, ok := v.(string); ok && s != "" {
				return s, true
			}
		}
	}
	return "", false
}
