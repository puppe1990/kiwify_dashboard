package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/puppe1990/cais/pkg/cais"
	"github.com/puppe1990/cais/pkg/cais/flash"
	"github.com/puppe1990/cais/pkg/cais/meta"
	"github.com/puppe1990/kiwify_dashboard/internal/kiwify"
	"github.com/puppe1990/kiwify_dashboard/internal/store"
	inertia "github.com/romsar/gonertia/v3"
)

const productsListPageSize = "50"

// ProductsAPI is the Kiwify products surface used by ProductsHandler (injectable for tests).
type ProductsAPI interface {
	ListProducts(ctx context.Context, q kiwify.PageQuery) (kiwify.ProductsPage, error)
	GetProduct(ctx context.Context, id string) (kiwify.Product, error)
}

// ProductsHandler serves read-only list and detail for Kiwify products.
type ProductsHandler struct {
	store     store.Store
	appSecret []byte
	site      meta.Site
	cfg       cais.Config
	inertia   *inertia.Inertia
	api       ProductsAPI // optional; if nil, built via NewClientFromStore
}

// NewProductsHandler constructs a ProductsHandler. api may be nil.
func NewProductsHandler(s store.Store, appSecret []byte, site meta.Site, cfg cais.Config, i *inertia.Inertia, api ProductsAPI) *ProductsHandler {
	return &ProductsHandler{store: s, appSecret: appSecret, site: site, cfg: cfg, inertia: i, api: api}
}

func (h *ProductsHandler) productsAPI() (ProductsAPI, error) {
	if h.api != nil {
		return h.api, nil
	}
	return kiwify.NewClientFromStore(h.store, h.appSecret, nil)
}

// List handles GET /products.
func (h *ProductsHandler) List(w http.ResponseWriter, r *http.Request) {
	props := inertia.Props{
		"site":     meta.ForRequest(h.site, r),
		"products": []any{},
		"pagination": map[string]any{
			"count":       0,
			"page_number": 1,
			"page_size":   50,
		},
		"errors": map[string]string{},
	}
	if msg, ok := flash.MessageFromRequest(r); ok {
		props["flash"] = inertia.Flash{msg.Kind: msg.Message}
	}

	api, err := h.productsAPI()
	if err != nil {
		props["errors"] = map[string]string{"client": "Credenciais Kiwify não configuradas ou inválidas."}
		_ = h.inertia.Render(w, r, "Products", props)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	page, err := api.ListProducts(ctx, kiwify.PageQuery{
		PageNumber: "1",
		PageSize:   productsListPageSize,
	})
	if err != nil {
		props["errors"] = map[string]string{"list": userFacingAPIError(err, "Não foi possível carregar produtos.")}
		_ = h.inertia.Render(w, r, "Products", props)
		return
	}

	props["products"] = page.Data
	props["pagination"] = map[string]any{
		"count":       page.Pagination.Count,
		"page_number": page.Pagination.PageNumber,
		"page_size":   page.Pagination.PageSize,
	}
	_ = h.inertia.Render(w, r, "Products", props)
}

// Show handles GET /products/{id}.
func (h *ProductsHandler) Show(w http.ResponseWriter, r *http.Request, id string) {
	props := inertia.Props{
		"site":    meta.ForRequest(h.site, r),
		"product": nil,
		"errors":  map[string]string{},
	}
	if msg, ok := flash.MessageFromRequest(r); ok {
		props["flash"] = inertia.Flash{msg.Kind: msg.Message}
	}

	api, err := h.productsAPI()
	if err != nil {
		props["errors"] = map[string]string{"client": "Credenciais Kiwify não configuradas ou inválidas."}
		_ = h.inertia.Render(w, r, "ProductShow", props)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	product, err := api.GetProduct(ctx, id)
	if err != nil {
		props["errors"] = map[string]string{"product": userFacingAPIError(err, "Não foi possível carregar o produto.")}
		_ = h.inertia.Render(w, r, "ProductShow", props)
		return
	}
	props["product"] = product
	_ = h.inertia.Render(w, r, "ProductShow", props)
}
