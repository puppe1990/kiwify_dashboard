package handlers

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/puppe1990/cais/pkg/cais"
	"github.com/puppe1990/cais/pkg/cais/httpx"
	"github.com/puppe1990/cais/pkg/cais/meta"
	"github.com/puppe1990/kiwify_dashboard/internal/crypto"
	"github.com/puppe1990/kiwify_dashboard/internal/kiwify"
	"github.com/puppe1990/kiwify_dashboard/internal/store"
	inertia "github.com/romsar/gonertia/v3"
)

type SetupHandler struct {
	store     store.Store
	appSecret []byte
	site      meta.Site
	cfg       cais.Config
	inertia   *inertia.Inertia
}

func NewSetupHandler(s store.Store, appSecret []byte, site meta.Site, cfg cais.Config, i *inertia.Inertia) *SetupHandler {
	return &SetupHandler{store: s, appSecret: appSecret, site: site, cfg: cfg, inertia: i}
}

func (h *SetupHandler) Get(w http.ResponseWriter, r *http.Request) {
	_ = h.inertia.Render(w, r, "Setup", inertia.Props{
		"site": meta.ForRequest(h.site, r),
	})
}

func (h *SetupHandler) Post(w http.ResponseWriter, r *http.Request) {
	if err := httpx.ParseFormOrJSON(r); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	clientID := strings.TrimSpace(r.FormValue("client_id"))
	clientSecret := strings.TrimSpace(r.FormValue("client_secret"))
	accountID := strings.TrimSpace(r.FormValue("account_id"))

	ve := inertia.ValidationErrors{}
	if clientID == "" {
		ve["client_id"] = "Client ID é obrigatório."
	}
	if clientSecret == "" {
		ve["client_secret"] = "Client Secret é obrigatório."
	}
	if accountID == "" {
		ve["account_id"] = "Account ID é obrigatório."
	}
	if len(ve) > 0 {
		ctx := inertia.SetValidationErrors(r.Context(), ve)
		_ = h.inertia.Render(w, r.WithContext(ctx), "Setup", inertia.Props{
			"site": meta.ForRequest(h.site, r),
		})
		return
	}

	ciphertext, err := crypto.Encrypt(h.appSecret, clientSecret)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := h.store.SaveKiwifySettings(store.KiwifySettings{
		AccountID:              accountID,
		ClientID:               clientID,
		ClientSecretCiphertext: ciphertext,
		// WebhookReceiveToken empty → store generates one
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	flash := inertia.Flash{"success": "Credenciais Kiwify salvas com sucesso."}
	// Optional OAuth smoke test — save already succeeded; warn only if token fails.
	if client, err := kiwify.NewClientFromStore(h.store, h.appSecret, nil); err == nil {
		ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
		defer cancel()
		if _, err := client.GetToken(ctx); err != nil {
			flash["warning"] = "Credenciais salvas, mas a autenticação OAuth com a Kiwify falhou. Verifique Client ID e Client Secret."
		}
	}

	ctx := inertia.SetFlash(r.Context(), flash)
	h.inertia.Redirect(w, r.WithContext(ctx), "/dashboard", http.StatusSeeOther)
}
