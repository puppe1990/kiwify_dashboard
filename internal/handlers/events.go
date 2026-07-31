package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/puppe1990/cais/pkg/cais"
	"github.com/puppe1990/cais/pkg/cais/flash"
	"github.com/puppe1990/cais/pkg/cais/meta"
	inertia "github.com/romsar/gonertia/v3"

	"github.com/puppe1990/kiwify_dashboard/internal/store"
)

const eventsPageSize = 50

// EventsHandler serves the local webhook events feed.
type EventsHandler struct {
	store   store.Store
	site    meta.Site
	cfg     cais.Config
	inertia *inertia.Inertia
}

// NewEventsHandler constructs an EventsHandler.
func NewEventsHandler(s store.Store, site meta.Site, cfg cais.Config, i *inertia.Inertia) *EventsHandler {
	return &EventsHandler{store: s, site: site, cfg: cfg, inertia: i}
}

// List handles GET /events — paginated local webhook_events.
func (h *EventsHandler) List(w http.ResponseWriter, r *http.Request) {
	pageNum := 1
	if p := r.URL.Query().Get("page"); p != "" {
		if n, err := strconv.Atoi(p); err == nil && n > 0 {
			pageNum = n
		}
	}
	offset := (pageNum - 1) * eventsPageSize

	props := inertia.Props{
		"site":   meta.ForRequest(h.site, r),
		"events": []any{},
		"pagination": map[string]any{
			"page":      pageNum,
			"page_size": eventsPageSize,
			"has_more":  false,
		},
		"errors": map[string]string{},
	}
	if msg, ok := flash.MessageFromRequest(r); ok {
		props["flash"] = inertia.Flash{msg.Kind: msg.Message}
	}

	// Fetch one extra to detect has_more without a COUNT query.
	events, err := h.store.ListWebhookEvents(eventsPageSize+1, offset)
	if err != nil {
		props["errors"] = map[string]string{"list": "Não foi possível carregar eventos de webhook."}
		_ = h.inertia.Render(w, r, "Events", props)
		return
	}

	hasMore := len(events) > eventsPageSize
	if hasMore {
		events = events[:eventsPageSize]
	}

	props["events"] = webhookEventsDetailForProps(events)
	props["pagination"] = map[string]any{
		"page":      pageNum,
		"page_size": eventsPageSize,
		"has_more":  hasMore,
	}
	_ = h.inertia.Render(w, r, "Events", props)
}

func webhookEventsDetailForProps(events []store.WebhookEvent) []map[string]any {
	out := make([]map[string]any, 0, len(events))
	for _, ev := range events {
		out = append(out, map[string]any{
			"id":          ev.ID,
			"eventType":   ev.EventType,
			"payloadJson": ev.PayloadJSON,
			"headersJson": ev.HeadersJSON,
			"receivedAt":  ev.ReceivedAt.UTC().Format(time.RFC3339),
			"processedOk": ev.ProcessedOK,
		})
	}
	return out
}
