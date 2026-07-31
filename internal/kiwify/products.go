package kiwify

import (
	"context"
	"fmt"
	"net/url"
)

// PageQuery is generic pagination for list endpoints.
type PageQuery struct {
	PageNumber string
	PageSize   string
}

func (q PageQuery) queryMap() map[string]string {
	m := map[string]string{}
	if q.PageNumber != "" {
		m["page_number"] = q.PageNumber
	}
	if q.PageSize != "" {
		m["page_size"] = q.PageSize
	}
	return m
}

// Product is a Kiwify product summary from GET /products.
type Product struct {
	ID               string   `json:"id"`
	Name             string   `json:"name"`
	Type             string   `json:"type"`
	Status           string   `json:"status"`
	Currency         string   `json:"currency"`
	Price            *float64 `json:"price"`
	AffiliateEnabled bool     `json:"affiliate_enabled"`
	PaymentType      string   `json:"payment_type"`
	CreatedAt        string   `json:"created_at"`
}

// ProductsPage is the GET /products response.
type ProductsPage struct {
	Data       []Product  `json:"data"`
	Pagination Pagination `json:"pagination"`
}

// ListProducts fetches a page of products.
func (c *Client) ListProducts(ctx context.Context, q PageQuery) (ProductsPage, error) {
	var out ProductsPage
	if err := c.GetJSON(ctx, "/products", q.queryMap(), &out); err != nil {
		return ProductsPage{}, err
	}
	if out.Data == nil {
		out.Data = []Product{}
	}
	return out, nil
}

// GetProduct fetches a single product by id.
func (c *Client) GetProduct(ctx context.Context, id string) (Product, error) {
	if id == "" {
		return Product{}, fmt.Errorf("kiwify: product id is required")
	}
	var out Product
	path := "/products/" + url.PathEscape(id)
	if err := c.GetJSON(ctx, path, nil, &out); err != nil {
		return Product{}, err
	}
	return out, nil
}
