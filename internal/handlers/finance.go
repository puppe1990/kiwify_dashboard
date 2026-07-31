package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"sync"
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

const payoutsListPageSize = "50"

// FinanceAPI is the Kiwify finance surface used by FinanceHandler (injectable for tests).
type FinanceAPI interface {
	ListBalances(ctx context.Context) (kiwify.Balances, error)
	ListPayouts(ctx context.Context, q kiwify.PageQuery) (kiwify.PayoutsPage, error)
	CreatePayout(ctx context.Context, amount float64) (kiwify.Payout, error)
}

// FinanceHandler serves balances, payouts list, and payout request.
type FinanceHandler struct {
	store     store.Store
	appSecret []byte
	site      meta.Site
	cfg       cais.Config
	inertia   *inertia.Inertia
	api       FinanceAPI // optional; if nil, built via NewClientFromStore
}

// NewFinanceHandler constructs a FinanceHandler. api may be nil.
func NewFinanceHandler(s store.Store, appSecret []byte, site meta.Site, cfg cais.Config, i *inertia.Inertia, api FinanceAPI) *FinanceHandler {
	return &FinanceHandler{store: s, appSecret: appSecret, site: site, cfg: cfg, inertia: i, api: api}
}

func (h *FinanceHandler) financeAPI() (FinanceAPI, error) {
	if h.api != nil {
		return h.api, nil
	}
	return kiwify.NewClientFromStore(h.store, h.appSecret, nil)
}

// Get handles GET /finance — balances + payouts list.
func (h *FinanceHandler) Get(w http.ResponseWriter, r *http.Request) {
	props := inertia.Props{
		"site":     meta.ForRequest(h.site, r),
		"balances": nil,
		"payouts":  []any{},
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

	api, err := h.financeAPI()
	if err != nil {
		props["errors"] = map[string]string{"client": "Credenciais Kiwify não configuradas ou inválidas."}
		_ = h.inertia.Render(w, r, "Finance", props)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	sectionErrs := map[string]string{}
	var (
		mu       sync.Mutex
		balances kiwify.Balances
		payouts  kiwify.PayoutsPage
		wg       sync.WaitGroup
	)

	wg.Add(2)
	go func() {
		defer wg.Done()
		b, err := api.ListBalances(ctx)
		if err != nil {
			mu.Lock()
			sectionErrs["balances"] = userFacingAPIError(err, "Não foi possível carregar saldos.")
			mu.Unlock()
			return
		}
		mu.Lock()
		balances = b
		mu.Unlock()
	}()
	go func() {
		defer wg.Done()
		page, err := api.ListPayouts(ctx, kiwify.PageQuery{
			PageNumber: "1",
			PageSize:   payoutsListPageSize,
		})
		if err != nil {
			mu.Lock()
			sectionErrs["payouts"] = userFacingAPIError(err, "Não foi possível carregar saques.")
			mu.Unlock()
			return
		}
		mu.Lock()
		payouts = page
		mu.Unlock()
	}()
	wg.Wait()

	if _, ok := sectionErrs["balances"]; !ok {
		props["balances"] = balances
	}
	if _, ok := sectionErrs["payouts"]; !ok {
		props["payouts"] = payouts.Data
		props["pagination"] = map[string]any{
			"count":       payouts.Pagination.Count,
			"page_number": payouts.Pagination.PageNumber,
			"page_size":   payouts.Pagination.PageSize,
		}
	}
	props["errors"] = sectionErrs
	_ = h.inertia.Render(w, r, "Finance", props)
}

// CreatePayout handles POST /finance/payouts. Always writes audit action finance.payout.
func (h *FinanceHandler) CreatePayout(w http.ResponseWriter, r *http.Request) {
	if err := httpx.ParseFormOrJSON(r); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	amountStr := strings.TrimSpace(r.FormValue("amount"))
	amount, err := strconv.ParseFloat(amountStr, 64)
	if err != nil || amount <= 0 {
		ctx := inertia.SetFlash(r.Context(), inertia.Flash{"error": "Informe um valor de saque válido."})
		h.inertia.Redirect(w, r.WithContext(ctx), "/finance", http.StatusSeeOther)
		return
	}

	userID, _ := session.UserID(r)
	ip := middleware.ClientIP(r, h.cfg)

	summary, _ := json.Marshal(map[string]any{
		"amount": amount,
	})

	respStatus := http.StatusOK
	respBody := `{"ok":true}`
	var payoutErr error
	var created kiwify.Payout

	api, err := h.financeAPI()
	if err != nil {
		payoutErr = err
		respStatus = http.StatusInternalServerError
		respBody = truncateAuditBody(err.Error())
	} else {
		ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
		defer cancel()
		created, payoutErr = api.CreatePayout(ctx, amount)
		if payoutErr != nil {
			if apiErr, ok := kiwify.AsAPIError(payoutErr); ok {
				respStatus = apiErr.Status
				if apiErr.Body != "" {
					respBody = truncateAuditBody(apiErr.Body)
				} else {
					respBody = truncateAuditBody(apiErr.Error())
				}
			} else {
				respStatus = http.StatusInternalServerError
				respBody = truncateAuditBody(payoutErr.Error())
			}
		} else if created.ID != "" {
			raw, _ := json.Marshal(map[string]any{"ok": true, "id": created.ID})
			respBody = string(raw)
		}
	}

	resourceID := created.ID
	if resourceID == "" {
		resourceID = amountStr
	}

	// Always write audit — success and failure.
	_, _ = h.store.InsertAuditLog(store.AuditLog{
		UserID:         userID,
		Action:         "finance.payout",
		ResourceType:   "payout",
		ResourceID:     resourceID,
		RequestSummary: string(summary),
		ResponseStatus: respStatus,
		ResponseBody:   respBody,
		IP:             ip,
	})

	flashKind := "success"
	flashMsg := "Saque solicitado com sucesso."
	if payoutErr != nil {
		flashKind = "error"
		flashMsg = userFacingAPIError(payoutErr, "Não foi possível solicitar o saque.")
	}

	ctx := inertia.SetFlash(r.Context(), inertia.Flash{flashKind: flashMsg})
	h.inertia.Redirect(w, r.WithContext(ctx), "/finance", http.StatusSeeOther)
}
