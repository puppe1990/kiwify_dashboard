package kiwify

import "context"

// Balances is the GET /balance response (available/pending balances for the account).
// Amounts are in minor units (centavos) as returned by the API.
type Balances struct {
	Available     float64 `json:"available"`
	Pending       float64 `json:"pending"`
	LegalEntityID string  `json:"legal_entity_id"`
}

// ListBalances fetches account balances from GET /balance.
func (c *Client) ListBalances(ctx context.Context) (Balances, error) {
	var out Balances
	if err := c.GetJSON(ctx, "/balance", nil, &out); err != nil {
		return Balances{}, err
	}
	return out, nil
}
