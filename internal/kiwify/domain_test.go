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
			"total_sales":                  3,
			"total_net_amount":             25956,
			"credit_card_approval_rate":    50,
			"refund_rate":                  25,
			"chargeback_rate":              0,
			"total_boleto_generated":       2,
			"total_boleto_paid":            1,
			"boleto_rate":                  50,
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
