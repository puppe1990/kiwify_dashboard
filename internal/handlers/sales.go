package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/puppe1990/cais/pkg/cais"
	"github.com/puppe1990/cais/pkg/cais/flash"
	"github.com/puppe1990/cais/pkg/cais/httpx"
	"github.com/puppe1990/cais/pkg/cais/meta"
	"github.com/puppe1990/cais/pkg/cais/middleware"
	"github.com/puppe1990/cais/pkg/cais/session"
	inertia "github.com/romsar/gonertia/v3"

	"github.com/puppe1990/kiwify_dashboard/internal/kiwify"
	"github.com/puppe1990/kiwify_dashboard/internal/store"
)

const (
	salesDateLayout   = "2006-01-02"
	salesMaxRangeDays = 90
	salesDefaultDays  = 30
	auditBodyMaxRunes = 2048
	salesListPageSize = "50"
)

// SalesAPI is the Kiwify sales surface used by SalesHandler (injectable for tests).
type SalesAPI interface {
	ListSales(ctx context.Context, q kiwify.SalesQuery) (kiwify.SalesPage, error)
	GetSale(ctx context.Context, id string) (kiwify.Sale, error)
	RefundSale(ctx context.Context, id string, pixKey string) error
}

// SalesHandler serves list, detail, and refund flows for Kiwify sales.
type SalesHandler struct {
	store     store.Store
	appSecret []byte
	site      meta.Site
	cfg       cais.Config
	inertia   *inertia.Inertia
	api       SalesAPI // optional; if nil, built via NewClientFromStore
}

// NewSalesHandler constructs a SalesHandler. api may be nil.
func NewSalesHandler(s store.Store, appSecret []byte, site meta.Site, cfg cais.Config, i *inertia.Inertia, api SalesAPI) *SalesHandler {
	return &SalesHandler{store: s, appSecret: appSecret, site: site, cfg: cfg, inertia: i, api: api}
}

func (h *SalesHandler) salesAPI() (SalesAPI, error) {
	if h.api != nil {
		return h.api, nil
	}
	return kiwify.NewClientFromStore(h.store, h.appSecret, nil)
}

// List handles GET /sales — default last 30 days; clamps range to max 90 days.
func (h *SalesHandler) List(w http.ResponseWriter, r *http.Request) {
	start, end := clampSalesDateRange(r.URL.Query().Get("start_date"), r.URL.Query().Get("end_date"), time.Now())

	props := inertia.Props{
		"site":      meta.ForRequest(h.site, r),
		"dateRange": map[string]string{"start": start, "end": end},
		"sales":     []any{},
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

	api, err := h.salesAPI()
	if err != nil {
		props["errors"] = map[string]string{"client": "Credenciais Kiwify não configuradas ou inválidas."}
		_ = h.inertia.Render(w, r, "Sales", props)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	page, err := api.ListSales(ctx, kiwify.SalesQuery{
		StartDate:  start,
		EndDate:    end,
		PageNumber: "1",
		PageSize:   salesListPageSize,
	})
	if err != nil {
		props["errors"] = map[string]string{"list": userFacingAPIError(err, "Não foi possível carregar vendas.")}
		_ = h.inertia.Render(w, r, "Sales", props)
		return
	}

	props["sales"] = page.Data
	props["pagination"] = map[string]any{
		"count":       page.Pagination.Count,
		"page_number": page.Pagination.PageNumber,
		"page_size":   page.Pagination.PageSize,
	}
	_ = h.inertia.Render(w, r, "Sales", props)
}

// Show handles GET /sales/{id}.
func (h *SalesHandler) Show(w http.ResponseWriter, r *http.Request, id string) {
	props := inertia.Props{
		"site":   meta.ForRequest(h.site, r),
		"sale":   nil,
		"errors": map[string]string{},
	}
	if msg, ok := flash.MessageFromRequest(r); ok {
		props["flash"] = inertia.Flash{msg.Kind: msg.Message}
	}

	api, err := h.salesAPI()
	if err != nil {
		props["errors"] = map[string]string{"client": "Credenciais Kiwify não configuradas ou inválidas."}
		_ = h.inertia.Render(w, r, "SaleShow", props)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	sale, err := api.GetSale(ctx, id)
	if err != nil {
		props["errors"] = map[string]string{"sale": userFacingAPIError(err, "Não foi possível carregar a venda.")}
		_ = h.inertia.Render(w, r, "SaleShow", props)
		return
	}
	props["sale"] = sale
	_ = h.inertia.Render(w, r, "SaleShow", props)
}

// Refund handles POST /sales/{id}/refund. Always writes an audit log (success and failure).
func (h *SalesHandler) Refund(w http.ResponseWriter, r *http.Request, id string) {
	if err := httpx.ParseFormOrJSON(r); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	pixKey := strings.TrimSpace(r.FormValue("pixKey"))
	if pixKey == "" {
		pixKey = strings.TrimSpace(r.FormValue("pix_key"))
	}

	userID, _ := session.UserID(r)
	ip := middleware.ClientIP(r, h.cfg)

	summary, _ := json.Marshal(map[string]any{
		"sale_id":     id,
		"has_pix_key": pixKey != "",
	})

	respStatus := http.StatusOK
	respBody := `{"ok":true}`
	var refundErr error

	api, err := h.salesAPI()
	if err != nil {
		refundErr = err
		respStatus = http.StatusInternalServerError
		respBody = truncateAuditBody(err.Error())
	} else {
		ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
		defer cancel()
		refundErr = api.RefundSale(ctx, id, pixKey)
		if refundErr != nil {
			if apiErr, ok := kiwify.AsAPIError(refundErr); ok {
				respStatus = apiErr.Status
				if apiErr.Body != "" {
					respBody = truncateAuditBody(apiErr.Body)
				} else {
					respBody = truncateAuditBody(apiErr.Error())
				}
			} else {
				respStatus = http.StatusInternalServerError
				respBody = truncateAuditBody(refundErr.Error())
			}
		}
	}

	// Always write audit — success and failure.
	_, _ = h.store.InsertAuditLog(store.AuditLog{
		UserID:         userID,
		Action:         "sales.refund",
		ResourceType:   "sale",
		ResourceID:     id,
		RequestSummary: string(summary),
		ResponseStatus: respStatus,
		ResponseBody:   respBody,
		IP:             ip,
	})

	flashKind := "success"
	flashMsg := "Reembolso solicitado com sucesso."
	if refundErr != nil {
		flashKind = "error"
		flashMsg = userFacingAPIError(refundErr, "Não foi possível reembolsar a venda.")
	}

	ctx := inertia.SetFlash(r.Context(), inertia.Flash{flashKind: flashMsg})
	h.inertia.Redirect(w, r.WithContext(ctx), "/sales/"+id, http.StatusSeeOther)
}

// clampSalesDateRange returns start/end as YYYY-MM-DD.
// Defaults to last 30 days; enforces start ≤ end and max 90-day window (clamps start).
func clampSalesDateRange(startStr, endStr string, now time.Time) (string, string) {
	// Normalize to date-only in local wall clock of now's location.
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	end := today
	start := today.AddDate(0, 0, -salesDefaultDays)

	if endStr != "" {
		if t, err := time.ParseInLocation(salesDateLayout, endStr, now.Location()); err == nil {
			end = t
		}
	}
	if startStr != "" {
		if t, err := time.ParseInLocation(salesDateLayout, startStr, now.Location()); err == nil {
			start = t
		}
	}

	if start.After(end) {
		start = end.AddDate(0, 0, -salesDefaultDays)
	}

	// Max 90-day inclusive window: if start is more than 90 days before end, pull start forward.
	// API docs: max 90-day window — clamp start to end - 90 days.
	minStart := end.AddDate(0, 0, -salesMaxRangeDays)
	if start.Before(minStart) {
		start = minStart
	}

	return start.Format(salesDateLayout), end.Format(salesDateLayout)
}

func truncateAuditBody(s string) string {
	if utf8.RuneCountInString(s) <= auditBodyMaxRunes {
		return s
	}
	runes := []rune(s)
	return string(runes[:auditBodyMaxRunes]) + "…"
}
