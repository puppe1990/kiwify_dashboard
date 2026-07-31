package handlers

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/puppe1990/cais/pkg/cais"
	"github.com/puppe1990/cais/pkg/cais/httpx"
	"github.com/puppe1990/cais/pkg/cais/meta"
	inertia "github.com/romsar/gonertia/v3"

	"github.com/puppe1990/kiwify_dashboard/internal/crypto"
	"github.com/puppe1990/kiwify_dashboard/internal/store"
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
	props := inertia.Props{
		"site": meta.ForRequest(h.site, r),
	}
	if f := flashProps(r); f != nil {
		props["flash"] = f
	}
	_ = h.inertia.Render(w, r, "Setup", props)
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

	// Clear any stale OAuth token so the next call fetches a fresh one.
	_ = h.store.UpdateOAuthToken("", time.Time{})

	kind, message := "success", "Credenciais salvas e validadas com a API Kiwify."
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	if err := probeKiwifyOAuth(ctx, h.store, h.appSecret); err != nil {
		kind = "error"
		message = "Credenciais salvas, mas a autenticação OAuth falhou: " + userFacingAPIError(err, "Client ID ou Client Secret inválidos.")
	}

	redirectWithFlash(w, r, h.inertia, h.cfg, kind, message, "/dashboard")
}
