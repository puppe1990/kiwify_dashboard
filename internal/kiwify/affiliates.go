package kiwify

import (
	"context"
	"fmt"
	"net/url"
)

// AffiliateProduct is the nested product summary on an affiliate.
type AffiliateProduct struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Affiliate is a Kiwify affiliate (list/detail).
// Commission is in minor units (centavos) as returned by the API.
// Note: the API uses affiliate_id rather than id.
type Affiliate struct {
	AffiliateID  string           `json:"affiliate_id"`
	Name         string           `json:"name"`
	Email        string           `json:"email"`
	CompanyName  string           `json:"company_name"`
	DirectorCPF  string           `json:"director_cpf"`
	CompanyCNPJ  string           `json:"company_cnpj"`
	Product      AffiliateProduct `json:"product"`
	Commission   float64          `json:"commission"`
	Status       string           `json:"status"`
	CreatedAt    string           `json:"created_at"`
}

// AffiliatesPage is the GET /affiliates response.
type AffiliatesPage struct {
	Data       []Affiliate `json:"data"`
	Pagination Pagination  `json:"pagination"`
}

// AffiliatesQuery filters GET /affiliates.
type AffiliatesQuery struct {
	PageNumber string
	PageSize   string
	Status     string
	ProductID  string
	Search     string
}

func (q AffiliatesQuery) queryMap() map[string]string {
	m := map[string]string{}
	if q.PageNumber != "" {
		m["page_number"] = q.PageNumber
	}
	if q.PageSize != "" {
		m["page_size"] = q.PageSize
	}
	if q.Status != "" {
		m["status"] = q.Status
	}
	if q.ProductID != "" {
		m["product_id"] = q.ProductID
	}
	if q.Search != "" {
		m["search"] = q.Search
	}
	return m
}

// ListAffiliates fetches a page of affiliates from GET /affiliates.
func (c *Client) ListAffiliates(ctx context.Context, q AffiliatesQuery) (AffiliatesPage, error) {
	var out AffiliatesPage
	if err := c.GetJSON(ctx, "/affiliates", q.queryMap(), &out); err != nil {
		return AffiliatesPage{}, err
	}
	if out.Data == nil {
		out.Data = []Affiliate{}
	}
	return out, nil
}

// GetAffiliate fetches a single affiliate by id from GET /affiliates/{id}.
func (c *Client) GetAffiliate(ctx context.Context, id string) (Affiliate, error) {
	if id == "" {
		return Affiliate{}, fmt.Errorf("kiwify: affiliate id is required")
	}
	var out Affiliate
	path := "/affiliates/" + url.PathEscape(id)
	if err := c.GetJSON(ctx, path, nil, &out); err != nil {
		return Affiliate{}, err
	}
	return out, nil
}

// UpdateAffiliate edits an affiliate via PUT /affiliates/{id}.
//
// Per Kiwify Public API docs (Editar afiliado), the request body accepts:
//   - commission (number)
//   - status (string: active | blocked | refused)
//
// body is map[string]any so callers can pass a subset of editable fields.
// Unknown extra keys are forwarded as-is to the API.
func (c *Client) UpdateAffiliate(ctx context.Context, id string, body map[string]any) (Affiliate, error) {
	if id == "" {
		return Affiliate{}, fmt.Errorf("kiwify: affiliate id is required")
	}
	if body == nil {
		body = map[string]any{}
	}
	var out Affiliate
	path := "/affiliates/" + url.PathEscape(id)
	if err := c.PutJSON(ctx, path, nil, body, &out); err != nil {
		return Affiliate{}, err
	}
	return out, nil
}
