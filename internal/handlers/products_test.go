package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/puppe1990/cais/pkg/cais"
	"github.com/puppe1990/cais/pkg/cais/middleware"

	"github.com/puppe1990/kiwify_dashboard/internal/kiwify"
	"github.com/puppe1990/kiwify_dashboard/internal/store"
)

type fakeProductsAPI struct {
	listFn    func(ctx context.Context, q kiwify.PageQuery) (kiwify.ProductsPage, error)
	getFn     func(ctx context.Context, id string) (kiwify.Product, error)
	lastQuery kiwify.PageQuery
	listN     int
	getN      int
	getID     string
}

func (f *fakeProductsAPI) ListProducts(ctx context.Context, q kiwify.PageQuery) (kiwify.ProductsPage, error) {
	f.listN++
	f.lastQuery = q
	if f.listFn != nil {
		return f.listFn(ctx, q)
	}
	return kiwify.ProductsPage{Data: []kiwify.Product{}}, nil
}

func (f *fakeProductsAPI) GetProduct(ctx context.Context, id string) (kiwify.Product, error) {
	f.getN++
	f.getID = id
	if f.getFn != nil {
		return f.getFn(ctx, id)
	}
	return kiwify.Product{ID: id, Name: "Produto", Status: "active"}, nil
}

func newProductsHandler(t *testing.T, api ProductsAPI) (*ProductsHandler, store.Store) {
	t.Helper()
	s := setupTestStore(t)
	h := NewProductsHandler(s, testAppSecret(), testSite(), cais.Config{}, setupTestInertia(t), api)
	return h, s
}

func TestProductsList_RendersProducts(t *testing.T) {
	price := 9900.0
	fake := &fakeProductsAPI{
		listFn: func(ctx context.Context, q kiwify.PageQuery) (kiwify.ProductsPage, error) {
			return kiwify.ProductsPage{
				Data: []kiwify.Product{
					{ID: "prod-1", Name: "Mentoria", Status: "active", Type: "membership", Price: &price},
				},
				Pagination: kiwify.Pagination{Count: 1, PageNumber: 1, PageSize: 50},
			}, nil
		},
	}
	h, _ := newProductsHandler(t, fake)

	req := inertiaRequest(http.MethodGet, "/products", nil)
	rr := httptest.NewRecorder()
	h.List(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rr.Code, rr.Body.String())
	}
	assertInertiaComponent(t, rr, "Products")
	products := assertInertiaProp(t, rr, "products").([]any)
	if len(products) != 1 {
		t.Fatalf("products len = %d", len(products))
	}
	p := products[0].(map[string]any)
	if p["id"] != "prod-1" || p["name"] != "Mentoria" {
		t.Fatalf("product = %v", p)
	}
	if fake.lastQuery.PageSize != "50" {
		t.Fatalf("page size = %q", fake.lastQuery.PageSize)
	}
	pag := assertInertiaProp(t, rr, "pagination").(map[string]any)
	if pag["count"] != float64(1) {
		t.Fatalf("pagination = %v", pag)
	}
}

func TestProductsList_APIError(t *testing.T) {
	fake := &fakeProductsAPI{
		listFn: func(ctx context.Context, q kiwify.PageQuery) (kiwify.ProductsPage, error) {
			return kiwify.ProductsPage{}, &kiwify.APIError{
				Status:      500,
				Message:     "boom",
				UserMessage: "Erro temporário da API Kiwify. Tente novamente.",
			}
		},
	}
	h, _ := newProductsHandler(t, fake)

	req := inertiaRequest(http.MethodGet, "/products", nil)
	rr := httptest.NewRecorder()
	h.List(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d", rr.Code)
	}
	assertInertiaComponent(t, rr, "Products")
	assertInertiaErrors(t, rr, "list")
}

func TestProductsShow_RendersProduct(t *testing.T) {
	fake := &fakeProductsAPI{
		getFn: func(ctx context.Context, id string) (kiwify.Product, error) {
			return kiwify.Product{ID: id, Name: "Curso X", Status: "active", Type: "course"}, nil
		},
	}
	h, _ := newProductsHandler(t, fake)

	req := inertiaRequest(http.MethodGet, "/products/prod-99", nil)
	rr := httptest.NewRecorder()
	h.Show(rr, req, "prod-99")

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d", rr.Code)
	}
	assertInertiaComponent(t, rr, "ProductShow")
	product := assertInertiaProp(t, rr, "product").(map[string]any)
	if product["id"] != "prod-99" || product["name"] != "Curso X" {
		t.Fatalf("product = %v", product)
	}
	if fake.getID != "prod-99" {
		t.Fatalf("getID = %s", fake.getID)
	}
}

func TestProductsShow_APIError(t *testing.T) {
	fake := &fakeProductsAPI{
		getFn: func(ctx context.Context, id string) (kiwify.Product, error) {
			return kiwify.Product{}, &kiwify.APIError{
				Status:      404,
				Message:     "not found",
				UserMessage: "Recurso não encontrado.",
			}
		},
	}
	h, _ := newProductsHandler(t, fake)

	req := inertiaRequest(http.MethodGet, "/products/missing", nil)
	rr := httptest.NewRecorder()
	h.Show(rr, req, "missing")

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d", rr.Code)
	}
	assertInertiaComponent(t, rr, "ProductShow")
	assertInertiaErrors(t, rr, "product")
}

func TestProductsList_RequiresAuth(t *testing.T) {
	fake := &fakeProductsAPI{}
	h, _ := newProductsHandler(t, fake)
	handler := middleware.RequireAuthFunc("/login", h.List)

	req := httptest.NewRequest(http.MethodGet, "/products", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusSeeOther || rr.Header().Get("Location") != "/login" {
		t.Fatalf("status=%d loc=%s", rr.Code, rr.Header().Get("Location"))
	}
	if fake.listN != 0 {
		t.Fatalf("ListProducts should not be called when unauthenticated")
	}
}

func TestProductsShow_RequiresAuth(t *testing.T) {
	fake := &fakeProductsAPI{}
	h, _ := newProductsHandler(t, fake)
	handler := middleware.RequireAuthFunc("/login", cais.StringParam("id", h.Show))

	req := httptest.NewRequest(http.MethodGet, "/products/p1", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusSeeOther || rr.Header().Get("Location") != "/login" {
		t.Fatalf("status=%d loc=%s", rr.Code, rr.Header().Get("Location"))
	}
	if fake.getN != 0 {
		t.Fatalf("GetProduct should not be called when unauthenticated")
	}
}
