package kiwify_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/puppe1990/kiwify_dashboard/internal/kiwify"
)

func testDomainClient(t *testing.T, handler http.HandlerFunc) *kiwify.Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/oauth/token" {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": "test-token",
				"expires_in":   3600,
			})
			return
		}
		handler(w, r)
	}))
	t.Cleanup(srv.Close)
	return kiwify.NewClient(kiwify.Config{
		BaseURL:      srv.URL + "/v1",
		AccountID:    "acc",
		ClientID:     "cid",
		ClientSecret: "sec",
		TokenStore:   &memTokenStore{},
		HTTPClient:   srv.Client(),
	})
}

func TestListSales_QueryParamsAndDecode(t *testing.T) {
	c := testDomainClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/sales" {
			t.Errorf("path = %s", r.URL.Path)
		}
		q := r.URL.Query()
		if q.Get("start_date") != "2024-01-01" || q.Get("end_date") != "2024-01-31" {
			t.Errorf("dates = %v", q)
		}
		if q.Get("page_number") != "1" || q.Get("page_size") != "10" {
			t.Errorf("pagination = %v", q)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"pagination": map[string]any{"count": 1, "page_number": 1, "page_size": 10},
			"data": []map[string]any{
				{
					"id":         "sale-1",
					"reference":  "REF123",
					"status":     "paid",
					"net_amount": 1500,
					"currency":   "BRL",
					"created_at": "2024-01-15T10:00:00.000Z",
					"product":    map[string]any{"id": "p1", "name": "Curso"},
					"customer":   map[string]any{"id": "c1", "name": "Ana", "email": "ana@ex.com"},
				},
			},
		})
	})

	page, err := c.ListSales(context.Background(), kiwify.SalesQuery{
		StartDate:  "2024-01-01",
		EndDate:    "2024-01-31",
		PageNumber: "1",
		PageSize:   "10",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Data) != 1 {
		t.Fatalf("len = %d", len(page.Data))
	}
	s := page.Data[0]
	if s.ID != "sale-1" || s.Reference != "REF123" || s.Status != "paid" {
		t.Fatalf("sale = %+v", s)
	}
	if s.NetAmount != 1500 || s.Product.Name != "Curso" {
		t.Fatalf("sale fields = %+v", s)
	}
	if page.Pagination.Count != 1 {
		t.Fatalf("pagination = %+v", page.Pagination)
	}
}

func TestGetSale(t *testing.T) {
	c := testDomainClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/sales/sale-99" {
			t.Errorf("path = %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": "sale-99", "reference": "X", "status": "paid", "created_at": "2024-01-01T00:00:00Z",
		})
	})
	sale, err := c.GetSale(context.Background(), "sale-99")
	if err != nil {
		t.Fatal(err)
	}
	if sale.ID != "sale-99" {
		t.Fatalf("got %+v", sale)
	}
}

func TestRefundSale_WithPixKey(t *testing.T) {
	c := testDomainClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s", r.Method)
		}
		if r.URL.Path != "/v1/sales/sale-1/refund" {
			t.Errorf("path = %s", r.URL.Path)
		}
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body["pixKey"] != "email@ex.com" {
			t.Fatalf("body = %v", body)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"refunded": true})
	})
	if err := c.RefundSale(context.Background(), "sale-1", "email@ex.com"); err != nil {
		t.Fatal(err)
	}
}

func TestRefundSale_WithoutPixKey(t *testing.T) {
	c := testDomainClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/sales/sale-2/refund" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if ct := r.Header.Get("Content-Type"); ct != "" {
			t.Errorf("expected no Content-Type without body, got %q", ct)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"refunded": true})
	})
	if err := c.RefundSale(context.Background(), "sale-2", ""); err != nil {
		t.Fatal(err)
	}
}

func TestSalesStats_QueryAndDecode(t *testing.T) {
	c := testDomainClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/stats" {
			t.Errorf("path = %s", r.URL.Path)
		}
		q := r.URL.Query()
		if q.Get("start_date") != "2024-06-01" || q.Get("end_date") != "2024-06-30" {
			t.Errorf("query = %v", q)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"total_sales":               3,
			"total_net_amount":          25956,
			"credit_card_approval_rate": 50,
			"refund_rate":               25,
			"chargeback_rate":           0,
			"total_boleto_generated":    2,
			"total_boleto_paid":         1,
			"boleto_rate":               50,
		})
	})

	stats, err := c.SalesStats(context.Background(), kiwify.StatsQuery{
		StartDate: "2024-06-01",
		EndDate:   "2024-06-30",
	})
	if err != nil {
		t.Fatal(err)
	}
	if stats.TotalSales != 3 || stats.TotalNetAmount != 25956 {
		t.Fatalf("stats = %+v", stats)
	}
	if stats.RefundRate != 25 || stats.BoletoRate != 50 {
		t.Fatalf("rates = %+v", stats)
	}
}

func TestListBalances(t *testing.T) {
	c := testDomainClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/balance" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("method = %s", r.Method)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"available":       236961,
			"pending":         4621,
			"legal_entity_id": "ent-1",
		})
	})

	bal, err := c.ListBalances(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if bal.Available != 236961 || bal.Pending != 4621 || bal.LegalEntityID != "ent-1" {
		t.Fatalf("balances = %+v", bal)
	}
}

func TestListProducts(t *testing.T) {
	c := testDomainClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/products" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if r.URL.Query().Get("page_size") != "20" {
			t.Errorf("query = %v", r.URL.Query())
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"pagination": map[string]any{"count": 1, "page_number": 1, "page_size": 20},
			"data": []map[string]any{
				{"id": "prod-1", "name": "Mentoria", "status": "active", "type": "membership"},
			},
		})
	})

	page, err := c.ListProducts(context.Background(), kiwify.PageQuery{PageSize: "20"})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Data) != 1 || page.Data[0].Name != "Mentoria" {
		t.Fatalf("products = %+v", page)
	}
}

func TestListPayouts(t *testing.T) {
	c := testDomainClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/payouts" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("method = %s", r.Method)
		}
		if r.URL.Query().Get("page_size") != "20" {
			t.Errorf("query = %v", r.URL.Query())
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"pagination": map[string]any{"count": 1, "page_number": 1, "page_size": 20},
			"data": []map[string]any{
				{"id": "po-1", "amount": 5000, "status": "paid", "created_at": "2024-01-01T00:00:00Z"},
			},
		})
	})

	page, err := c.ListPayouts(context.Background(), kiwify.PageQuery{PageSize: "20"})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Data) != 1 || page.Data[0].ID != "po-1" || page.Data[0].Amount != 5000 {
		t.Fatalf("payouts = %+v", page)
	}
}

func TestGetPayout(t *testing.T) {
	c := testDomainClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/payouts/po-99" {
			t.Errorf("path = %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": "po-99", "amount": 1200, "status": "pending",
		})
	})
	p, err := c.GetPayout(context.Background(), "po-99")
	if err != nil {
		t.Fatal(err)
	}
	if p.ID != "po-99" || p.Amount != 1200 {
		t.Fatalf("payout = %+v", p)
	}
}

func TestCreatePayout(t *testing.T) {
	c := testDomainClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s", r.Method)
		}
		if r.URL.Path != "/v1/payouts/" {
			t.Errorf("path = %s", r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if body["amount"] != float64(5000) {
			t.Fatalf("body = %v", body)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"id": "new-po"})
	})
	p, err := c.CreatePayout(context.Background(), 5000)
	if err != nil {
		t.Fatal(err)
	}
	if p.ID != "new-po" {
		t.Fatalf("payout = %+v", p)
	}
	if p.Amount != 5000 {
		t.Fatalf("amount filled = %v", p.Amount)
	}
}

func TestListAffiliates(t *testing.T) {
	c := testDomainClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/affiliates" {
			t.Errorf("path = %s", r.URL.Path)
		}
		q := r.URL.Query()
		if q.Get("page_size") != "10" || q.Get("status") != "active" {
			t.Errorf("query = %v", q)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"pagination": map[string]any{"count": 1, "page_number": 1, "page_size": 10},
			"data": []map[string]any{
				{
					"affiliate_id": "aff-1",
					"name":         "MY Affiliate",
					"email":        "myaffiliate@mail.com",
					"status":       "active",
					"commission":   4600,
					"product":      map[string]any{"id": "p1", "name": "My Product"},
				},
			},
		})
	})

	page, err := c.ListAffiliates(context.Background(), kiwify.AffiliatesQuery{
		PageSize: "10",
		Status:   "active",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Data) != 1 {
		t.Fatalf("len = %d", len(page.Data))
	}
	a := page.Data[0]
	if a.AffiliateID != "aff-1" || a.Name != "MY Affiliate" || a.Commission != 4600 {
		t.Fatalf("affiliate = %+v", a)
	}
	if a.Product.Name != "My Product" {
		t.Fatalf("product = %+v", a.Product)
	}
}

func TestGetAffiliate(t *testing.T) {
	c := testDomainClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/affiliates/aff-99" {
			t.Errorf("path = %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"affiliate_id": "aff-99",
			"name":         "João",
			"status":       "active",
			"commission":   1200,
		})
	})
	a, err := c.GetAffiliate(context.Background(), "aff-99")
	if err != nil {
		t.Fatal(err)
	}
	if a.AffiliateID != "aff-99" || a.Name != "João" {
		t.Fatalf("affiliate = %+v", a)
	}
}

func TestUpdateAffiliate(t *testing.T) {
	c := testDomainClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("method = %s", r.Method)
		}
		if r.URL.Path != "/v1/affiliates/aff-1" {
			t.Errorf("path = %s", r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if body["status"] != "blocked" || body["commission"] != float64(5000) {
			t.Fatalf("body = %v", body)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"affiliate_id": "aff-1",
			"status":       "blocked",
			"commission":   5000,
			"name":         "MY Affiliate",
		})
	})
	a, err := c.UpdateAffiliate(context.Background(), "aff-1", map[string]any{
		"status":     "blocked",
		"commission": 5000,
	})
	if err != nil {
		t.Fatal(err)
	}
	if a.Status != "blocked" || a.Commission != 5000 {
		t.Fatalf("affiliate = %+v", a)
	}
}

func TestListWebhooks(t *testing.T) {
	c := testDomainClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s", r.Method)
		}
		if r.URL.Path != "/v1/webhooks" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if r.URL.Query().Get("page_size") != "50" {
			t.Errorf("query = %v", r.URL.Query())
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"pagination": map[string]any{"count": 1, "page_number": 1, "page_size": 50},
			"data": []map[string]any{
				{
					"id":       "wh-1",
					"name":     "ops",
					"url":      "https://example.com/hook",
					"products": "all",
					"triggers": []string{"compra_aprovada"},
					"token":    "tok1",
				},
			},
		})
	})

	page, err := c.ListWebhooks(context.Background(), kiwify.WebhooksQuery{PageSize: "50"})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Data) != 1 {
		t.Fatalf("len = %d", len(page.Data))
	}
	wh := page.Data[0]
	if wh.ID != "wh-1" || wh.Name != "ops" || wh.Products != "all" {
		t.Fatalf("webhook = %+v", wh)
	}
	if len(wh.Triggers) != 1 || wh.Triggers[0] != "compra_aprovada" {
		t.Fatalf("triggers = %v", wh.Triggers)
	}
}

func TestCreateWebhook(t *testing.T) {
	c := testDomainClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s", r.Method)
		}
		if r.URL.Path != "/v1/webhooks" {
			t.Errorf("path = %s", r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if body["name"] != "meu webhook" {
			t.Fatalf("name = %v", body["name"])
		}
		if body["url"] != "https://example.com/webhooks/kiwify/abc" {
			t.Fatalf("url = %v", body["url"])
		}
		if body["products"] != "all" {
			t.Fatalf("products = %v", body["products"])
		}
		triggers, ok := body["triggers"].([]any)
		if !ok || len(triggers) != 2 {
			t.Fatalf("triggers = %v", body["triggers"])
		}
		if body["token"] != "secret-token" {
			t.Fatalf("token = %v", body["token"])
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":       "wh-new",
			"name":     body["name"],
			"url":      body["url"],
			"products": body["products"],
			"triggers": body["triggers"],
			"token":    body["token"],
		})
	})

	wh, err := c.CreateWebhook(context.Background(), kiwify.WebhookInput{
		Name:     "meu webhook",
		URL:      "https://example.com/webhooks/kiwify/abc",
		Products: "all",
		Triggers: []string{"compra_aprovada", "compra_reembolsada"},
		Token:    "secret-token",
	})
	if err != nil {
		t.Fatal(err)
	}
	if wh.ID != "wh-new" || wh.Name != "meu webhook" {
		t.Fatalf("webhook = %+v", wh)
	}
}

func TestGetWebhook(t *testing.T) {
	c := testDomainClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/webhooks/wh-99" {
			t.Errorf("path = %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": "wh-99", "name": "detail", "url": "https://x", "products": "prod-1",
			"triggers": []string{"pix_gerado"},
		})
	})
	wh, err := c.GetWebhook(context.Background(), "wh-99")
	if err != nil {
		t.Fatal(err)
	}
	if wh.ID != "wh-99" || wh.Products != "prod-1" {
		t.Fatalf("webhook = %+v", wh)
	}
}

func TestUpdateWebhook(t *testing.T) {
	c := testDomainClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("method = %s", r.Method)
		}
		if r.URL.Path != "/v1/webhooks/wh-1" {
			t.Errorf("path = %s", r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if body["name"] != "updated" {
			t.Fatalf("body = %v", body)
		}
		// optional token omitted → not present
		if _, ok := body["token"]; ok {
			t.Fatalf("token should be omitted when empty, body=%v", body)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": "wh-1", "name": "updated", "url": body["url"], "products": body["products"],
			"triggers": body["triggers"],
		})
	})
	wh, err := c.UpdateWebhook(context.Background(), "wh-1", kiwify.WebhookInput{
		Name:     "updated",
		URL:      "https://example.com/h",
		Products: "all",
		Triggers: []string{"chargeback"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if wh.Name != "updated" {
		t.Fatalf("webhook = %+v", wh)
	}
}

func TestDeleteWebhook(t *testing.T) {
	c := testDomainClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("method = %s", r.Method)
		}
		if r.URL.Path != "/v1/webhooks/wh-del" {
			t.Errorf("path = %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	if err := c.DeleteWebhook(context.Background(), "wh-del"); err != nil {
		t.Fatal(err)
	}
}

func TestGetAccount_FlatObject(t *testing.T) {
	c := testDomainClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/account" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("method = %s", r.Method)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":    "acc-1",
			"name":  "Kiwify Store",
			"email": "owner@store.com",
			"plan":  "pro",
		})
	})
	acc, err := c.GetAccount(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if acc.ID != "acc-1" || acc.Name != "Kiwify Store" || acc.Email != "owner@store.com" {
		t.Fatalf("account = %+v", acc)
	}
	if acc.Raw["plan"] != "pro" {
		t.Fatalf("raw = %v", acc.Raw)
	}
}

func TestGetAccount_WrappedData(t *testing.T) {
	c := testDomainClient(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"id":           "acc-2",
				"company_name": "Wrapped Co",
				"owner_email":  "a@b.com",
			},
		})
	})
	acc, err := c.GetAccount(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if acc.ID != "acc-2" {
		t.Fatalf("id = %q", acc.ID)
	}
	if acc.Name != "Wrapped Co" {
		t.Fatalf("name = %q", acc.Name)
	}
	if acc.Email != "a@b.com" {
		t.Fatalf("email = %q", acc.Email)
	}
}
