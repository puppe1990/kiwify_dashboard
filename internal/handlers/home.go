package handlers

import (
	"fmt"
	"net/http"

	"github.com/puppe1990/cais/pkg/cais"
	"github.com/puppe1990/cais/pkg/cais/flash"
	"github.com/puppe1990/cais/pkg/cais/i18n"
	"github.com/puppe1990/cais/pkg/cais/meta"
	"github.com/puppe1990/cais/pkg/cais/session"
	inertia "github.com/romsar/gonertia/v3"

	"github.com/puppe1990/kiwify_dashboard/internal/store"
)

type HomeHandler struct {
	renderer *cais.Renderer
	store    store.Store
	site     meta.Site
	catalog  *i18n.Catalog
	cfg      cais.Config
	inertia  *inertia.Inertia
}

func NewHomeHandler(renderer *cais.Renderer, site meta.Site, catalog *i18n.Catalog, cfg cais.Config, i *inertia.Inertia) *HomeHandler {
	return &HomeHandler{renderer: renderer, site: site, catalog: catalog, cfg: cfg, inertia: i}
}

// SetStore attaches the store for auth redirect decisions (configured → dashboard).
// Optional for tests that only render the public home page.
func (h *HomeHandler) SetStore(s store.Store) {
	h.store = s
}

func NewHomeHandlerWithStore(renderer *cais.Renderer, s store.Store, site meta.Site, catalog *i18n.Catalog, cfg cais.Config, i *inertia.Inertia) *HomeHandler {
	return &HomeHandler{renderer: renderer, store: s, site: site, catalog: catalog, cfg: cfg, inertia: i}
}

func (h *HomeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Authenticated users leave the public landing for ops pages.
	if _, ok := session.UserID(r); ok {
		target := "/dashboard"
		if h.store != nil {
			configured, err := h.store.Configured()
			if err == nil && !configured {
				target = "/setup"
			}
		}
		h.inertia.Redirect(w, r, target, http.StatusSeeOther)
		return
	}

	site := meta.ForRequest(h.site, r)
	props := inertia.Props{
		"title": h.catalog.T("home.title"),
		"site":  site,
		"labels": map[string]string{
			"heading":   h.catalog.T("home.rails_heading"),
			"subtitle":  fmt.Sprintf(h.catalog.T("home.rails_subtitle"), site.AppName),
			"stack":     h.catalog.T("home.stack"),
			"contact":   h.catalog.T("home.contact_link"),
			"login":     h.catalog.T("auth.login_submit"),
			"dashboard": h.catalog.T("dashboard.title"),
		},
	}
	if msg, ok := flash.MessageFromRequest(r); ok {
		props["flash"] = inertia.Flash{msg.Kind: msg.Message}
	}
	if err := h.inertia.Render(w, r, "Home", props); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
