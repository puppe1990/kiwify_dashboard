package handlers

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/puppe1990/cais/pkg/cais"
	"github.com/puppe1990/cais/pkg/cais/flash"
	"github.com/puppe1990/cais/pkg/cais/meta"
	"github.com/puppe1990/kiwify_dashboard/internal/kiwify"
	"github.com/puppe1990/kiwify_dashboard/internal/store"
	inertia "github.com/romsar/gonertia/v3"
)

type DashboardHandler struct {
	renderer  *cais.Renderer
	store     store.Store
	appSecret []byte
	site      meta.Site
	cfg       cais.Config
	inertia   *inertia.Inertia
}

func NewDashboardHandler(renderer *cais.Renderer, s store.Store, appSecret []byte, site meta.Site, cfg cais.Config, i *inertia.Inertia) *DashboardHandler {
	return &DashboardHandler{renderer: renderer, store: s, appSecret: appSecret, site: site, cfg: cfg, inertia: i}
}

func (h *DashboardHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	now := time.Now()
	end := now.Format("2006-01-02")
	start := now.AddDate(0, 0, -30).Format("2006-01-02")
	dateRange := map[string]string{"start": start, "end": end}

	props := inertia.Props{
		"site":      meta.ForRequest(h.site, r),
		"dateRange": dateRange,
		"stats":     nil,
		"balances":  nil,
		"sales":     []any{},
		"events":    []any{},
		"errors":    map[string]string{},
	}
	if msg, ok := flash.MessageFromRequest(r); ok {
		props["flash"] = inertia.Flash{msg.Kind: msg.Message}
	}

	sectionErrs := map[string]string{}

	events, err := h.store.ListWebhookEvents(10, 0)
	if err != nil {
		sectionErrs["events"] = "Não foi possível carregar eventos de webhook."
	} else {
		props["events"] = webhookEventsForProps(events)
	}

	client, err := kiwify.NewClientFromStore(h.store, h.appSecret, nil)
	if err != nil {
		// Middleware should ensure setup; still render a usable page.
		sectionErrs["client"] = "Credenciais Kiwify não configuradas ou inválidas."
		props["errors"] = sectionErrs
		_ = h.inertia.Render(w, r, "Dashboard", props)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	var (
		mu       sync.Mutex
		stats    kiwify.Stats
		balances kiwify.Balances
		sales    kiwify.SalesPage
		wg       sync.WaitGroup
	)

	setErr := func(key, msg string) {
		mu.Lock()
		sectionErrs[key] = msg
		mu.Unlock()
	}

	wg.Add(3)
	go func() {
		defer wg.Done()
		s, err := client.SalesStats(ctx, kiwify.StatsQuery{StartDate: start, EndDate: end})
		if err != nil {
			setErr("stats", userFacingAPIError(err, "Não foi possível carregar estatísticas de vendas."))
			return
		}
		mu.Lock()
		stats = s
		mu.Unlock()
	}()
	go func() {
		defer wg.Done()
		b, err := client.ListBalances(ctx)
		if err != nil {
			setErr("balances", userFacingAPIError(err, "Não foi possível carregar saldos."))
			return
		}
		mu.Lock()
		balances = b
		mu.Unlock()
	}()
	go func() {
		defer wg.Done()
		page, err := client.ListSales(ctx, kiwify.SalesQuery{
			StartDate:  start,
			EndDate:    end,
			PageNumber: "1",
			PageSize:   "10",
		})
		if err != nil {
			setErr("sales", userFacingAPIError(err, "Não foi possível carregar vendas recentes."))
			return
		}
		mu.Lock()
		sales = page
		mu.Unlock()
	}()
	wg.Wait()

	if _, ok := sectionErrs["stats"]; !ok {
		props["stats"] = stats
	}
	if _, ok := sectionErrs["balances"]; !ok {
		props["balances"] = balances
	}
	if _, ok := sectionErrs["sales"]; !ok {
		props["sales"] = sales.Data
	}
	props["errors"] = sectionErrs

	_ = h.inertia.Render(w, r, "Dashboard", props)
}

func userFacingAPIError(err error, fallback string) string {
	if apiErr, ok := kiwify.AsAPIError(err); ok && apiErr.UserMessage != "" {
		return apiErr.UserMessage
	}
	return fallback
}

func webhookEventsForProps(events []store.WebhookEvent) []map[string]any {
	out := make([]map[string]any, 0, len(events))
	for _, ev := range events {
		out = append(out, map[string]any{
			"id":          ev.ID,
			"eventType":   ev.EventType,
			"receivedAt":  ev.ReceivedAt.UTC().Format(time.RFC3339),
			"processedOk": ev.ProcessedOK,
		})
	}
	return out
}
