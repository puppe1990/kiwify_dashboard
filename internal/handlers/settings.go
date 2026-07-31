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

func (h *SettingsHandler) settingsProps(r *http.Request, settings store.KiwifySettings, apiStatus string, apiStatusMessage string) inertia.Props {
	base := strings.TrimRight(h.cfg.AppURL, "/")
	if base == "" {
		base = strings.TrimRight(h.site.AppURL, "/")
	}
	webhookURL := ""
	if settings.WebhookReceiveToken != "" {
		webhookURL = base + "/webhooks/kiwify/" + settings.WebhookReceiveToken
	}
	props := inertia.Props{
		"accountId":         settings.AccountID,
		"clientId":          settings.ClientID,
		"hasSecret":         settings.ClientSecretCiphertext != "",
		"webhookReceiveURL": webhookURL,
		"apiStatus":         apiStatus,
		"apiStatusMessage":  apiStatusMessage,
		"site":              meta.ForRequest(h.site, r),
	}
	if f := flashProps(r); f != nil {
		props["flash"] = f
	}
	return props
}

// apiStatusFromSettings derives UI status without calling the network.
func apiStatusFromSettings(settings store.KiwifySettings) (status, message string) {
	if settings.AccountID == "" || settings.ClientID == "" || settings.ClientSecretCiphertext == "" {
		return "missing", "Credenciais incompletas. Preencha Account ID, Client ID e Client Secret."
	}
	if settings.OAuthAccessTokenCiphertext != "" && settings.TokenExpiresAt != nil && settings.TokenExpiresAt.After(time.Now().Add(30*time.Second)) {
		return "ok", "Token OAuth válido em cache — última validação com a Kiwify bem-sucedida."
	}
	if settings.OAuthAccessTokenCiphertext != "" && settings.TokenExpiresAt != nil && !settings.TokenExpiresAt.After(time.Now()) {
		return "expired", "Token OAuth expirado. Salve as configurações novamente para revalidar."
	}
	return "untested", "Credenciais salvas, mas ainda não há token OAuth validado. Clique em “Salvar” ou “Testar conexão”."
}

func (h *SettingsHandler) Get(w http.ResponseWriter, r *http.Request) {
	settings, err := h.store.GetKiwifySettings()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	status, msg := apiStatusFromSettings(settings)
	_ = h.inertia.Render(w, r, "Settings", h.settingsProps(r, settings, status, msg))
}

func (h *SettingsHandler) Post(w http.ResponseWriter, r *http.Request) {
	if err := httpx.ParseFormOrJSON(r); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	clientID := strings.TrimSpace(r.FormValue("client_id"))
	clientSecret := strings.TrimSpace(r.FormValue("client_secret"))
	accountID := strings.TrimSpace(r.FormValue("account_id"))
	// test_only=1 revalidates without requiring a full form change (optional).
	testOnly := r.FormValue("test_only") == "1" || r.FormValue("intent") == "test"

	ve := inertia.ValidationErrors{}
	if !testOnly {
		if clientID == "" {
			ve["client_id"] = "Client ID é obrigatório."
		}
		if accountID == "" {
			ve["account_id"] = "Account ID é obrigatório."
		}
	}
	if len(ve) > 0 {
		settings, _ := h.store.GetKiwifySettings()
		status, msg := apiStatusFromSettings(settings)
		ctx := inertia.SetValidationErrors(r.Context(), ve)
		_ = h.inertia.Render(w, r.WithContext(ctx), "Settings", h.settingsProps(r, settings, status, msg))
		return
	}

	existing, err := h.store.GetKiwifySettings()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if !testOnly {
		secretCipher := existing.ClientSecretCiphertext
		if clientSecret != "" {
			ciphertext, err := crypto.Encrypt(h.appSecret, clientSecret)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			secretCipher = ciphertext
		}

		credsChanged := clientID != existing.ClientID || accountID != existing.AccountID || clientSecret != ""

		// Preserve webhook token (empty WebhookReceiveToken keeps existing).
		// Clear OAuth cache when credentials change so status reflects a real re-test.
		oauthCipher := existing.OAuthAccessTokenCiphertext
		var tokenExp *time.Time
		tokenExp = existing.TokenExpiresAt
		if credsChanged {
			oauthCipher = ""
			tokenExp = nil
		}

		if err := h.store.SaveKiwifySettings(store.KiwifySettings{
			AccountID:                  accountID,
			ClientID:                   clientID,
			ClientSecretCiphertext:     secretCipher,
			OAuthAccessTokenCiphertext: oauthCipher,
			TokenExpiresAt:             tokenExp,
			WebhookReceiveToken:        "", // empty → store preserves existing
		}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if credsChanged {
			_ = h.store.UpdateOAuthToken("", time.Time{})
		}
	}

	kind, message := h.testOAuthConnection(r.Context())
	redirectWithFlash(w, r, h.inertia, h.cfg, kind, message, "/settings")
}

func (h *SettingsHandler) testOAuthConnection(parent context.Context) (kind, message string) {
	ctx, cancel := context.WithTimeout(parent, 10*time.Second)
	defer cancel()
	if err := probeKiwifyOAuth(ctx, h.store, h.appSecret); err != nil {
		return "error", "Falha ao autenticar na API Kiwify: " + userFacingAPIError(err, "Client ID ou Client Secret inválidos.")
	}
	return "success", "Conexão OK — token OAuth obtido com sucesso da API Kiwify."
}
