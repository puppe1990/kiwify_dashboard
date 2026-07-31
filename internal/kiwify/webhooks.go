package kiwify

import (
	"context"
	"fmt"
	"net/url"
)

// Known webhook trigger values from the Kiwify Public API.
var WebhookTriggers = []string{
	"boleto_gerado",
	"pix_gerado",
	"carrinho_abandonado",
	"compra_recusada",
	"compra_aprovada",
	"compra_reembolsada",
	"chargeback",
	"subscription_canceled",
	"subscription_late",
	"subscription_renewed",
}

// Webhook is a Kiwify webhook registration (list/detail/create/update response).
type Webhook struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	URL       string   `json:"url"`
	Products  string   `json:"products"` // "all" or a product id
	Triggers  []string `json:"triggers"`
	Token     string   `json:"token"`
	CreatedAt string   `json:"created_at"`
	UpdatedAt string   `json:"updated_at"`
}

// WebhooksPage is the GET /webhooks response.
type WebhooksPage struct {
	Data       []Webhook  `json:"data"`
	Pagination Pagination `json:"pagination"`
}

// WebhooksQuery filters GET /webhooks.
type WebhooksQuery struct {
	PageNumber string
	PageSize   string
	ProductID  string
	Search     string
}

func (q WebhooksQuery) queryMap() map[string]string {
	m := map[string]string{}
	if q.PageNumber != "" {
		m["page_number"] = q.PageNumber
	}
	if q.PageSize != "" {
		m["page_size"] = q.PageSize
	}
	if q.ProductID != "" {
		m["product_id"] = q.ProductID
	}
	if q.Search != "" {
		m["search"] = q.Search
	}
	return m
}

// WebhookInput is the body for create/update webhook.
// Products is "all" or a product id. Token is optional.
type WebhookInput struct {
	Name     string   `json:"name"`
	URL      string   `json:"url"`
	Products string   `json:"products"`
	Triggers []string `json:"triggers"`
	Token    string   `json:"token,omitempty"`
}

func (in WebhookInput) toBody() map[string]any {
	triggers := in.Triggers
	if triggers == nil {
		triggers = []string{}
	}
	body := map[string]any{
		"name":     in.Name,
		"url":      in.URL,
		"products": in.Products,
		"triggers": triggers,
	}
	if in.Token != "" {
		body["token"] = in.Token
	}
	return body
}

// ListWebhooks fetches a page of webhooks from GET /webhooks.
func (c *Client) ListWebhooks(ctx context.Context, q WebhooksQuery) (WebhooksPage, error) {
	var out WebhooksPage
	if err := c.GetJSON(ctx, "/webhooks", q.queryMap(), &out); err != nil {
		return WebhooksPage{}, err
	}
	if out.Data == nil {
		out.Data = []Webhook{}
	}
	return out, nil
}

// GetWebhook fetches a single webhook by id from GET /webhooks/{id}.
func (c *Client) GetWebhook(ctx context.Context, id string) (Webhook, error) {
	if id == "" {
		return Webhook{}, fmt.Errorf("kiwify: webhook id is required")
	}
	var out Webhook
	path := "/webhooks/" + url.PathEscape(id)
	if err := c.GetJSON(ctx, path, nil, &out); err != nil {
		return Webhook{}, err
	}
	if out.Triggers == nil {
		out.Triggers = []string{}
	}
	return out, nil
}

// CreateWebhook creates a webhook via POST /webhooks.
func (c *Client) CreateWebhook(ctx context.Context, in WebhookInput) (Webhook, error) {
	var out Webhook
	if err := c.PostJSON(ctx, "/webhooks", nil, in.toBody(), &out); err != nil {
		return Webhook{}, err
	}
	if out.Triggers == nil {
		out.Triggers = []string{}
	}
	return out, nil
}

// UpdateWebhook updates a webhook via PUT /webhooks/{id}.
func (c *Client) UpdateWebhook(ctx context.Context, id string, in WebhookInput) (Webhook, error) {
	if id == "" {
		return Webhook{}, fmt.Errorf("kiwify: webhook id is required")
	}
	var out Webhook
	path := "/webhooks/" + url.PathEscape(id)
	if err := c.PutJSON(ctx, path, nil, in.toBody(), &out); err != nil {
		return Webhook{}, err
	}
	if out.Triggers == nil {
		out.Triggers = []string{}
	}
	return out, nil
}

// DeleteWebhook deletes a webhook via DELETE /webhooks/{id}.
func (c *Client) DeleteWebhook(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("kiwify: webhook id is required")
	}
	path := "/webhooks/" + url.PathEscape(id)
	return c.DeleteJSON(ctx, path, nil, nil)
}
