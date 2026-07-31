package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/puppe1990/cais/pkg/cais"
	"github.com/puppe1990/cais/pkg/cais/flash"
	"github.com/puppe1990/cais/pkg/cais/httpx"
	"github.com/puppe1990/cais/pkg/cais/meta"
	"github.com/puppe1990/cais/pkg/cais/middleware"
	"github.com/puppe1990/cais/pkg/cais/session"
	"github.com/puppe1990/kiwify_dashboard/internal/kiwify"
	"github.com/puppe1990/kiwify_dashboard/internal/store"
	inertia "github.com/romsar/gonertia/v3"
)

const affiliatesListPageSize = "50"

// AffiliatesAPI is the Kiwify affiliates surface used by AffiliatesHandler (injectable for tests).
type AffiliatesAPI interface {
	ListAffiliates(ctx context.Context, q kiwify.AffiliatesQuery) (kiwify.AffiliatesPage, error)
	GetAffiliate(ctx context.Context, id string) (kiwify.Affiliate, error)
	UpdateAffiliate(ctx context.Context, id string, body map[string]any) (kiwify.Affiliate, error)
}

// AffiliatesHandler serves list, detail, and edit flows for Kiwify affiliates.
type AffiliatesHandler struct {
	store     store.Store
	appSecret []byte
	site      meta.Site
	cfg       cais.Config
	inertia   *inertia.Inertia
	api       AffiliatesAPI // optional; if nil, built via NewClientFromStore
}

// NewAffiliatesHandler constructs an AffiliatesHandler. api may be nil.
func NewAffiliatesHandler(s store.Store, appSecret []byte, site meta.Site, cfg cais.Config, i *inertia.Inertia, api AffiliatesAPI) *AffiliatesHandler {
	return &AffiliatesHandler{store: s, appSecret: appSecret, site: site, cfg: cfg, inertia: i, api: api}
}

func (h *AffiliatesHandler) affiliatesAPI() (AffiliatesAPI, error) {
	if h.api != nil {
		return h.api, nil
	}
	return kiwify.NewClientFromStore(h.store, h.appSecret, nil)
}

// List handles GET /affiliates.
func (h *AffiliatesHandler) List(w http.ResponseWriter, r *http.Request) {
	props := inertia.Props{
		"site":       meta.ForRequest(h.site, r),
		"affiliates": []any{},
		"pagination": map[string]any{
			"count":       0,
			"page_number": 1,
			"page_size":   50,
		},
		"filters": map[string]string{
			"status":     strings.TrimSpace(r.URL.Query().Get("status")),
			"search":     strings.TrimSpace(r.URL.Query().Get("search")),
			"product_id": strings.TrimSpace(r.URL.Query().Get("product_id")),
		},
		"errors": map[string]string{},
	}
	if msg, ok := flash.MessageFromRequest(r); ok {
		props["flash"] = inertia.Flash{msg.Kind: msg.Message}
	}

	api, err := h.affiliatesAPI()
	if err != nil {
		props["errors"] = map[string]string{"client": "Credenciais Kiwify não configuradas ou inválidas."}
		_ = h.inertia.Render(w, r, "Affiliates", props)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	filters := props["filters"].(map[string]string)
	page, err := api.ListAffiliates(ctx, kiwify.AffiliatesQuery{
		PageNumber: "1",
		PageSize:   affiliatesListPageSize,
		Status:     filters["status"],
		Search:     filters["search"],
		ProductID:  filters["product_id"],
	})
	if err != nil {
		props["errors"] = map[string]string{"list": userFacingAPIError(err, "Não foi possível carregar afiliados.")}
		_ = h.inertia.Render(w, r, "Affiliates", props)
		return
	}

	props["affiliates"] = page.Data
	props["pagination"] = map[string]any{
		"count":       page.Pagination.Count,
		"page_number": page.Pagination.PageNumber,
		"page_size":   page.Pagination.PageSize,
	}
	_ = h.inertia.Render(w, r, "Affiliates", props)
}

// Show handles GET /affiliates/{id}.
func (h *AffiliatesHandler) Show(w http.ResponseWriter, r *http.Request, id string) {
	props := inertia.Props{
		"site":      meta.ForRequest(h.site, r),
		"affiliate": nil,
		"errors":    map[string]string{},
	}
	if msg, ok := flash.MessageFromRequest(r); ok {
		props["flash"] = inertia.Flash{msg.Kind: msg.Message}
	}

	api, err := h.affiliatesAPI()
	if err != nil {
		props["errors"] = map[string]string{"client": "Credenciais Kiwify não configuradas ou inválidas."}
		_ = h.inertia.Render(w, r, "AffiliateShow", props)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	aff, err := api.GetAffiliate(ctx, id)
	if err != nil {
		props["errors"] = map[string]string{"affiliate": userFacingAPIError(err, "Não foi possível carregar o afiliado.")}
		_ = h.inertia.Render(w, r, "AffiliateShow", props)
		return
	}
	props["affiliate"] = aff
	_ = h.inertia.Render(w, r, "AffiliateShow", props)
}

// Update handles POST /affiliates/{id} (edit form). Always audits affiliates.edit.
func (h *AffiliatesHandler) Update(w http.ResponseWriter, r *http.Request, id string) {
	if err := httpx.ParseFormOrJSON(r); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	status := strings.TrimSpace(r.FormValue("status"))
	commissionStr := strings.TrimSpace(r.FormValue("commission"))

	body := map[string]any{}
	if status != "" {
		body["status"] = status
	}
	if commissionStr != "" {
		// Commission is in API minor units (centavos), matching GET affiliate payloads.
		comm, err := strconv.ParseFloat(strings.ReplaceAll(commissionStr, ",", "."), 64)
		if err != nil || comm < 0 {
			ctx := inertia.SetFlash(r.Context(), inertia.Flash{"error": "Informe uma comissão válida."})
			h.inertia.Redirect(w, r.WithContext(ctx), "/affiliates/"+id, http.StatusSeeOther)
			return
		}
		body["commission"] = comm
	}

	if len(body) == 0 {
		ctx := inertia.SetFlash(r.Context(), inertia.Flash{"error": "Informe ao menos status ou comissão para atualizar."})
		h.inertia.Redirect(w, r.WithContext(ctx), "/affiliates/"+id, http.StatusSeeOther)
		return
	}

	userID, _ := session.UserID(r)
	ip := middleware.ClientIP(r, h.cfg)

	summary, _ := json.Marshal(map[string]any{
		"affiliate_id": id,
		"body":         body,
	})

	respStatus := http.StatusOK
	respBody := `{"ok":true}`
	var updateErr error

	api, err := h.affiliatesAPI()
	if err != nil {
		updateErr = err
		respStatus = http.StatusInternalServerError
		respBody = truncateAuditBody(err.Error())
	} else {
		ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
		defer cancel()
		updated, err := api.UpdateAffiliate(ctx, id, body)
		updateErr = err
		if updateErr != nil {
			if apiErr, ok := kiwify.AsAPIError(updateErr); ok {
				respStatus = apiErr.Status
				if apiErr.Body != "" {
					respBody = truncateAuditBody(apiErr.Body)
				} else {
					respBody = truncateAuditBody(apiErr.Error())
				}
			} else {
				respStatus = http.StatusInternalServerError
				respBody = truncateAuditBody(updateErr.Error())
			}
		} else if updated.AffiliateID != "" {
			raw, _ := json.Marshal(map[string]any{"ok": true, "affiliate_id": updated.AffiliateID})
			respBody = string(raw)
		}
	}

	// Always write audit — success and failure.
	_, _ = h.store.InsertAuditLog(store.AuditLog{
		UserID:         userID,
		Action:         "affiliates.edit",
		ResourceType:   "affiliate",
		ResourceID:     id,
		RequestSummary: string(summary),
		ResponseStatus: respStatus,
		ResponseBody:   respBody,
		IP:             ip,
	})

	flashKind := "success"
	flashMsg := "Afiliado atualizado com sucesso."
	if updateErr != nil {
		flashKind = "error"
		flashMsg = userFacingAPIError(updateErr, "Não foi possível atualizar o afiliado.")
	}

	ctx := inertia.SetFlash(r.Context(), inertia.Flash{flashKind: flashMsg})
	h.inertia.Redirect(w, r.WithContext(ctx), "/affiliates/"+id, http.StatusSeeOther)
}
