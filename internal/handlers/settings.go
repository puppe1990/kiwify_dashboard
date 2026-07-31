package handlers

import (
	"net/http"
	"strings"

	"github.com/puppe1990/cais/pkg/cais"
	"github.com/puppe1990/cais/pkg/cais/httpx"
	"github.com/puppe1990/cais/pkg/cais/meta"
	"github.com/puppe1990/kiwify_dashboard/internal/crypto"
	"github.com/puppe1990/kiwify_dashboard/internal/store"
	inertia "github.com/romsar/gonertia/v3"
)

type SettingsHandler struct {
	store     store.Store
	appSecret []byte
	site      meta.Site
	cfg       cais.Config
	inertia   *inertia.Inertia
}

func NewSettingsHandler(s store.Store, appSecret []byte, site meta.Site, cfg cais.Config, i *inertia.Inertia) *SettingsHandler {
	return &SettingsHandler{store: s, appSecret: appSecret, site: site, cfg: cfg, inertia: i}
}

func (h *SettingsHandler) settingsProps(r *http.Request, settings store.KiwifySettings) inertia.Props {
	base := strings.TrimRight(h.cfg.AppURL, "/")
	if base == "" {
		base = strings.TrimRight(h.site.AppURL, "/")
	}
	webhookURL := ""
	if settings.WebhookReceiveToken != "" {
		webhookURL = base + "/webhooks/kiwify/" + settings.WebhookReceiveToken
	}
	return inertia.Props{
		"accountId":         settings.AccountID,
		"clientId":          settings.ClientID,
		"hasSecret":         settings.ClientSecretCiphertext != "",
		"webhookReceiveURL": webhookURL,
		"site":              meta.ForRequest(h.site, r),
	}
}

func (h *SettingsHandler) Get(w http.ResponseWriter, r *http.Request) {
	settings, err := h.store.GetKiwifySettings()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_ = h.inertia.Render(w, r, "Settings", h.settingsProps(r, settings))
}

func (h *SettingsHandler) Post(w http.ResponseWriter, r *http.Request) {
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
	if accountID == "" {
		ve["account_id"] = "Account ID é obrigatório."
	}
	if len(ve) > 0 {
		settings, _ := h.store.GetKiwifySettings()
		ctx := inertia.SetValidationErrors(r.Context(), ve)
		_ = h.inertia.Render(w, r.WithContext(ctx), "Settings", h.settingsProps(r, settings))
		return
	}

	existing, err := h.store.GetKiwifySettings()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	secretCipher := existing.ClientSecretCiphertext
	if clientSecret != "" {
		ciphertext, err := crypto.Encrypt(h.appSecret, clientSecret)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		secretCipher = ciphertext
	}

	// Preserve OAuth token, expiry, and webhook token (empty WebhookReceiveToken keeps existing).
	if err := h.store.SaveKiwifySettings(store.KiwifySettings{
		AccountID:                  accountID,
		ClientID:                   clientID,
		ClientSecretCiphertext:     secretCipher,
		OAuthAccessTokenCiphertext: existing.OAuthAccessTokenCiphertext,
		TokenExpiresAt:             existing.TokenExpiresAt,
		WebhookReceiveToken:        "", // empty → store preserves existing
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	ctx := inertia.SetFlash(r.Context(), inertia.Flash{"success": "Configurações atualizadas."})
	h.inertia.Redirect(w, r.WithContext(ctx), "/settings", http.StatusSeeOther)
}
