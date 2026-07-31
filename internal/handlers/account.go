package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/puppe1990/cais/pkg/cais"
	"github.com/puppe1990/cais/pkg/cais/flash"
	"github.com/puppe1990/cais/pkg/cais/meta"
	"github.com/puppe1990/kiwify_dashboard/internal/kiwify"
	"github.com/puppe1990/kiwify_dashboard/internal/store"
	inertia "github.com/romsar/gonertia/v3"
)

// AccountAPI is the Kiwify account surface used by AccountHandler (injectable for tests).
type AccountAPI interface {
	GetAccount(ctx context.Context) (kiwify.Account, error)
}

// AccountHandler serves account details from the Kiwify Public API.
type AccountHandler struct {
	store     store.Store
	appSecret []byte
	site      meta.Site
	cfg       cais.Config
	inertia   *inertia.Inertia
	api       AccountAPI // optional; if nil, built via NewClientFromStore
}

// NewAccountHandler constructs an AccountHandler. api may be nil.
func NewAccountHandler(s store.Store, appSecret []byte, site meta.Site, cfg cais.Config, i *inertia.Inertia, api AccountAPI) *AccountHandler {
	return &AccountHandler{store: s, appSecret: appSecret, site: site, cfg: cfg, inertia: i, api: api}
}

func (h *AccountHandler) accountAPI() (AccountAPI, error) {
	if h.api != nil {
		return h.api, nil
	}
	return kiwify.NewClientFromStore(h.store, h.appSecret, nil)
}

// Get handles GET /account.
func (h *AccountHandler) Get(w http.ResponseWriter, r *http.Request) {
	props := inertia.Props{
		"site":    meta.ForRequest(h.site, r),
		"account": nil,
		"errors":  map[string]string{},
	}
	if msg, ok := flash.MessageFromRequest(r); ok {
		props["flash"] = inertia.Flash{msg.Kind: msg.Message}
	}

	api, err := h.accountAPI()
	if err != nil {
		props["errors"] = map[string]string{"client": "Credenciais Kiwify não configuradas ou inválidas."}
		_ = h.inertia.Render(w, r, "Account", props)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	account, err := api.GetAccount(ctx)
	if err != nil {
		props["errors"] = map[string]string{"account": userFacingAPIError(err, "Não foi possível carregar os dados da conta.")}
		_ = h.inertia.Render(w, r, "Account", props)
		return
	}

	props["account"] = accountForProps(account)
	_ = h.inertia.Render(w, r, "Account", props)
}

func accountForProps(a kiwify.Account) map[string]any {
	raw := a.Raw
	if raw == nil {
		raw = map[string]any{}
	}
	return map[string]any{
		"id":          a.ID,
		"name":        a.Name,
		"email":       a.Email,
		"companyName": a.CompanyName,
		"directorCpf": a.DirectorCPF,
		"companyCnpj": a.CompanyCNPJ,
		"raw":         raw,
	}
}
