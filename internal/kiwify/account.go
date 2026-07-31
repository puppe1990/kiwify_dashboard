package kiwify

import (
	"context"
	"encoding/json"
)

// Account is a view of GET /account-details (Kiwify Public API).
// Known fields match the official schema; Raw keeps the full payload for UI.
type Account struct {
	ID          string `json:"id"`
	Name        string `json:"name"` // filled from company_name when present
	Email       string `json:"email"`
	CompanyName string `json:"company_name"`
	DirectorCPF string `json:"director_cpf"`
	CompanyCNPJ string `json:"company_cnpj"`
	// Raw is the full decoded JSON object (or wrapper data) for unknown fields.
	Raw map[string]any `json:"-"`
}

// GetAccount fetches account details from GET /account-details.
// Response shapes vary; we accept a flat object or a { "data": {...} } wrapper.
func (c *Client) GetAccount(ctx context.Context) (Account, error) {
	var raw json.RawMessage
	if err := c.GetJSON(ctx, "/account-details", nil, &raw); err != nil {
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

	// Fallback field names from the official AccountDetails schema.
	if acc.ID == "" {
		if v, ok := stringFromMap(m, "account_id", "accountId"); ok {
			acc.ID = v
		}
	}
	if acc.CompanyName == "" {
		if v, ok := stringFromMap(m, "company_name", "companyName"); ok {
			acc.CompanyName = v
		}
	}
	if acc.Name == "" {
		if acc.CompanyName != "" {
			acc.Name = acc.CompanyName
		} else if v, ok := stringFromMap(m, "full_name", "fullName"); ok {
			acc.Name = v
		}
	}
	if acc.Email == "" {
		if v, ok := stringFromMap(m, "owner_email", "ownerEmail"); ok {
			acc.Email = v
		}
	}
	if acc.DirectorCPF == "" {
		if v, ok := stringFromMap(m, "director_cpf", "directorCpf"); ok {
			acc.DirectorCPF = v
		}
	}
	if acc.CompanyCNPJ == "" {
		if v, ok := stringFromMap(m, "company_cnpj", "companyCnpj"); ok {
			acc.CompanyCNPJ = v
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
