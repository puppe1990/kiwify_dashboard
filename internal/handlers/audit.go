package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/puppe1990/cais/pkg/cais"
	"github.com/puppe1990/cais/pkg/cais/flash"
	"github.com/puppe1990/cais/pkg/cais/meta"
	"github.com/puppe1990/kiwify_dashboard/internal/store"
	inertia "github.com/romsar/gonertia/v3"
)

const auditPageSize = 50

// AuditHandler serves the local audit log of sensitive actions.
type AuditHandler struct {
	store   store.Store
	site    meta.Site
	cfg     cais.Config
	inertia *inertia.Inertia
}

// NewAuditHandler constructs an AuditHandler.
func NewAuditHandler(s store.Store, site meta.Site, cfg cais.Config, i *inertia.Inertia) *AuditHandler {
	return &AuditHandler{store: s, site: site, cfg: cfg, inertia: i}
}

// List handles GET /audit — paginated audit_logs from the store.
func (h *AuditHandler) List(w http.ResponseWriter, r *http.Request) {
	pageNum := 1
	if p := r.URL.Query().Get("page"); p != "" {
		if n, err := strconv.Atoi(p); err == nil && n > 0 {
			pageNum = n
		}
	}
	offset := (pageNum - 1) * auditPageSize

	props := inertia.Props{
		"site":  meta.ForRequest(h.site, r),
		"logs":  []any{},
		"pagination": map[string]any{
			"page":      pageNum,
			"page_size": auditPageSize,
			"has_more":  false,
		},
		"errors": map[string]string{},
	}
	if msg, ok := flash.MessageFromRequest(r); ok {
		props["flash"] = inertia.Flash{msg.Kind: msg.Message}
	}

	// Fetch one extra to detect has_more without a COUNT query.
	logs, err := h.store.ListAuditLogs(auditPageSize+1, offset)
	if err != nil {
		props["errors"] = map[string]string{"list": "Não foi possível carregar o log de auditoria."}
		_ = h.inertia.Render(w, r, "Audit", props)
		return
	}

	hasMore := len(logs) > auditPageSize
	if hasMore {
		logs = logs[:auditPageSize]
	}

	props["logs"] = auditLogsForProps(logs)
	props["pagination"] = map[string]any{
		"page":      pageNum,
		"page_size": auditPageSize,
		"has_more":  hasMore,
	}
	_ = h.inertia.Render(w, r, "Audit", props)
}

func auditLogsForProps(logs []store.AuditLog) []map[string]any {
	out := make([]map[string]any, 0, len(logs))
	for _, l := range logs {
		out = append(out, map[string]any{
			"id":             l.ID,
			"userId":         l.UserID,
			"action":         l.Action,
			"resourceType":   l.ResourceType,
			"resourceId":     l.ResourceID,
			"requestSummary": l.RequestSummary,
			"responseStatus": l.ResponseStatus,
			"ip":             l.IP,
			"createdAt":      l.CreatedAt.UTC().Format(time.RFC3339),
		})
	}
	return out
}
