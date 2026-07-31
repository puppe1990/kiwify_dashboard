package kiwify

import (
	"context"
	"fmt"
	"net/url"
)

// SalesQuery filters GET /sales.
// start_date and end_date are required by the API (max 90-day window).
type SalesQuery struct {
	StartDate  string
	EndDate    string
	PageNumber string
	PageSize   string
	Status     string
	ProductID  string
}

func (q SalesQuery) queryMap() map[string]string {
	m := map[string]string{}
	if q.StartDate != "" {
		m["start_date"] = q.StartDate
	}
	if q.EndDate != "" {
		m["end_date"] = q.EndDate
	}
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
	return m
}

// SaleProduct is the nested product summary on a sale.
type SaleProduct struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// SaleCustomer is the nested customer summary on a sale.
type SaleCustomer struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// Sale is a Kiwify sale (list/detail). Secondary nested fields may be empty.
// Monetary amounts are in minor units (centavos) as returned by the API.
type Sale struct {
	ID            string       `json:"id"`
	Reference     string       `json:"reference"`
	Type          string       `json:"type"`
	Status        string       `json:"status"`
	PaymentMethod string       `json:"payment_method"`
	NetAmount     float64      `json:"net_amount"`
	Currency      string       `json:"currency"`
	CreatedAt     string       `json:"created_at"`
	UpdatedAt     string       `json:"updated_at"`
	Product       SaleProduct  `json:"product"`
	Customer      SaleCustomer `json:"customer"`
}

// Pagination is the common list pagination envelope.
type Pagination struct {
	Count      float64 `json:"count"`
	PageNumber float64 `json:"page_number"`
	PageSize   float64 `json:"page_size"`
}

// SalesPage is the GET /sales response.
type SalesPage struct {
	Data       []Sale     `json:"data"`
	Pagination Pagination `json:"pagination"`
}

// ListSales fetches a page of sales for the given query.
func (c *Client) ListSales(ctx context.Context, q SalesQuery) (SalesPage, error) {
	var out SalesPage
	if err := c.GetJSON(ctx, "/sales", q.queryMap(), &out); err != nil {
		return SalesPage{}, err
	}
	if out.Data == nil {
		out.Data = []Sale{}
	}
	return out, nil
}

// GetSale fetches a single sale by id.
func (c *Client) GetSale(ctx context.Context, id string) (Sale, error) {
	if id == "" {
		return Sale{}, fmt.Errorf("kiwify: sale id is required")
	}
	var out Sale
	path := "/sales/" + url.PathEscape(id)
	if err := c.GetJSON(ctx, path, nil, &out); err != nil {
		return Sale{}, err
	}
	return out, nil
}

// RefundSale requests a refund for the sale. When pixKey is non-empty, it is
// sent as JSON body {"pixKey":"..."}; otherwise the request has no body.
func (c *Client) RefundSale(ctx context.Context, id string, pixKey string) error {
	if id == "" {
		return fmt.Errorf("kiwify: sale id is required")
	}
	path := "/sales/" + url.PathEscape(id) + "/refund"
	var body any
	if pixKey != "" {
		body = map[string]string{"pixKey": pixKey}
	}
	return c.PostJSON(ctx, path, nil, body, nil)
}
