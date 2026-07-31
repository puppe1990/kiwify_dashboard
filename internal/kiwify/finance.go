package kiwify

import (
	"context"
	"fmt"
	"net/url"
)

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

// Payout is a Kiwify payout (list/detail).
// Amount is in minor units (centavos) as returned by the API.
type Payout struct {
	ID        string  `json:"id"`
	Amount    float64 `json:"amount"`
	Status    string  `json:"status"`
	Currency  string  `json:"currency"`
	CreatedAt string  `json:"created_at"`
	UpdatedAt string  `json:"updated_at"`
}

// PayoutsPage is the GET /payouts response.
type PayoutsPage struct {
	Data       []Payout   `json:"data"`
	Pagination Pagination `json:"pagination"`
}

// ListPayouts fetches a page of payouts from GET /payouts.
func (c *Client) ListPayouts(ctx context.Context, q PageQuery) (PayoutsPage, error) {
	var out PayoutsPage
	if err := c.GetJSON(ctx, "/payouts", q.queryMap(), &out); err != nil {
		return PayoutsPage{}, err
	}
	if out.Data == nil {
		out.Data = []Payout{}
	}
	return out, nil
}

// GetPayout fetches a single payout by id from GET /payouts/{id}.
func (c *Client) GetPayout(ctx context.Context, id string) (Payout, error) {
	if id == "" {
		return Payout{}, fmt.Errorf("kiwify: payout id is required")
	}
	var out Payout
	path := "/payouts/" + url.PathEscape(id)
	if err := c.GetJSON(ctx, path, nil, &out); err != nil {
		return Payout{}, err
	}
	return out, nil
}

// CreatePayout requests a payout via POST /payouts/ with body {"amount": amount}.
// Amount is in the same unit expected by the API (typically centavos / minor units).
// Returns the created payout id when the API includes one in the response.
func (c *Client) CreatePayout(ctx context.Context, amount float64) (Payout, error) {
	var out Payout
	body := map[string]any{"amount": amount}
	// Trailing slash matches Kiwify Public API docs: POST /v1/payouts/
	if err := c.PostJSON(ctx, "/payouts/", nil, body, &out); err != nil {
		return Payout{}, err
	}
	if out.Amount == 0 {
		out.Amount = amount
	}
	return out, nil
}
