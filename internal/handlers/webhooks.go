package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
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

const webhooksListPageSize = "50"

// WebhooksAPI is the Kiwify webhooks surface used by WebhooksHandler (injectable for tests).
type WebhooksAPI interface {
	ListWebhooks(ctx context.Context, q kiwify.WebhooksQuery) (kiwify.WebhooksPage, error)
	GetWebhook(ctx context.Context, id string) (kiwify.Webhook, error)
	CreateWebhook(ctx context.Context, in kiwify.WebhookInput) (kiwify.Webhook, error)
	UpdateWebhook(ctx context.Context, id string, in kiwify.WebhookInput) (kiwify.Webhook, error)
	DeleteWebhook(ctx context.Context, id string) error
}

// WebhooksHandler serves list/create/edit/delete for Kiwify webhooks.
type WebhooksHandler struct {
	store     store.Store
	appSecret []byte
	site      meta.Site
	cfg       cais.Config
	inertia   *inertia.Inertia
	api       WebhooksAPI // optional; if nil, built via NewClientFromStore
}

// NewWebhooksHandler constructs a WebhooksHandler. api may be nil.
func NewWebhooksHandler(s store.Store, appSecret []byte, site meta.Site, cfg cais.Config, i *inertia.Inertia, api WebhooksAPI) *WebhooksHandler {
	return &WebhooksHandler{store: s, appSecret: appSecret, site: site, cfg: cfg, inertia: i, api: api}
}

func (h *WebhooksHandler) webhooksAPI() (WebhooksAPI, error) {
	if h.api != nil {
		return h.api, nil
	}
	return kiwify.NewClientFromStore(h.store, h.appSecret, nil)
}

func (h *WebhooksHandler) defaultReceiveURL() string {
	settings, err := h.store.GetKiwifySettings()
	if err != nil || settings.WebhookReceiveToken == "" {
		return ""
	}
	base := strings.TrimRight(h.cfg.AppURL, "/")
	if base == "" {
		base = strings.TrimRight(h.site.AppURL, "/")
	}
	if base == "" {
		return ""
	}
	return base + "/webhooks/kiwify/" + settings.WebhookReceiveToken
}

// List handles GET /webhooks.
func (h *WebhooksHandler) List(w http.ResponseWriter, r *http.Request) {
	props := inertia.Props{
		"site":     meta.ForRequest(h.site, r),
		"webhooks": []any{},
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

	api, err := h.webhooksAPI()
	if err != nil {
		props["errors"] = map[string]string{"client": "Credenciais Kiwify não configuradas ou inválidas."}
		_ = h.inertia.Render(w, r, "Webhooks", props)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	page, err := api.ListWebhooks(ctx, kiwify.WebhooksQuery{
		PageNumber: "1",
		PageSize:   webhooksListPageSize,
	})
	if err != nil {
		props["errors"] = map[string]string{"list": userFacingAPIError(err, "Não foi possível carregar webhooks.")}
		_ = h.inertia.Render(w, r, "Webhooks", props)
		return
	}

	props["webhooks"] = page.Data
	props["pagination"] = map[string]any{
		"count":       page.Pagination.Count,
		"page_number": page.Pagination.PageNumber,
		"page_size":   page.Pagination.PageSize,
	}
	_ = h.inertia.Render(w, r, "Webhooks", props)
}

// New handles GET /webhooks/new — create form prefilled with receive URL.
func (h *WebhooksHandler) New(w http.ResponseWriter, r *http.Request) {
	props := inertia.Props{
		"site":       meta.ForRequest(h.site, r),
		"webhook":    nil,
		"triggers":   kiwify.WebhookTriggers,
		"defaultUrl": h.defaultReceiveURL(),
		"errors":     map[string]string{},
	}
	if msg, ok := flash.MessageFromRequest(r); ok {
		props["flash"] = inertia.Flash{msg.Kind: msg.Message}
	}
	_ = h.inertia.Render(w, r, "WebhookForm", props)
}

// Create handles POST /webhooks. Audits webhooks.create.
func (h *WebhooksHandler) Create(w http.ResponseWriter, r *http.Request) {
	if err := httpx.ParseFormOrJSON(r); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	in, ve := parseWebhookForm(r)
	if len(ve) > 0 {
		ctx := inertia.SetValidationErrors(r.Context(), ve)
		props := inertia.Props{
			"site":       meta.ForRequest(h.site, r),
			"webhook":    nil,
			"triggers":   kiwify.WebhookTriggers,
			"defaultUrl": h.defaultReceiveURL(),
			"form": map[string]any{
				"name":     in.Name,
				"url":      in.URL,
				"products": in.Products,
				"triggers": in.Triggers,
				"token":    in.Token,
			},
			"errors": map[string]string{},
		}
		_ = h.inertia.Render(w, r.WithContext(ctx), "WebhookForm", props)
		return
	}

	userID, _ := session.UserID(r)
	ip := middleware.ClientIP(r, h.cfg)
	summary, _ := json.Marshal(map[string]any{
		"name":     in.Name,
		"url":      in.URL,
		"products": in.Products,
		"triggers": in.Triggers,
	})

	respStatus := http.StatusOK
	respBody := `{"ok":true}`
	var createErr error
	var created kiwify.Webhook

	api, err := h.webhooksAPI()
	if err != nil {
		createErr = err
		respStatus = http.StatusInternalServerError
		respBody = truncateAuditBody(err.Error())
	} else {
		ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
		defer cancel()
		created, createErr = api.CreateWebhook(ctx, in)
		if createErr != nil {
			if apiErr, ok := kiwify.AsAPIError(createErr); ok {
				respStatus = apiErr.Status
				if apiErr.Body != "" {
					respBody = truncateAuditBody(apiErr.Body)
				} else {
					respBody = truncateAuditBody(apiErr.Error())
				}
			} else {
				respStatus = http.StatusInternalServerError
				respBody = truncateAuditBody(createErr.Error())
			}
		} else if created.ID != "" {
			raw, _ := json.Marshal(map[string]any{"ok": true, "id": created.ID})
			respBody = string(raw)
		}
	}

	resourceID := created.ID
	_, _ = h.store.InsertAuditLog(store.AuditLog{
		UserID:         userID,
		Action:         "webhooks.create",
		ResourceType:   "webhook",
		ResourceID:     resourceID,
		RequestSummary: string(summary),
		ResponseStatus: respStatus,
		ResponseBody:   respBody,
		IP:             ip,
	})

	if createErr != nil {
		ctx := inertia.SetFlash(r.Context(), inertia.Flash{
			"error": userFacingAPIError(createErr, "Não foi possível criar o webhook."),
		})
		h.inertia.Redirect(w, r.WithContext(ctx), "/webhooks/new", http.StatusSeeOther)
		return
	}

	ctx := inertia.SetFlash(r.Context(), inertia.Flash{"success": "Webhook criado com sucesso."})
	redirect := "/webhooks"
	if created.ID != "" {
		redirect = "/webhooks/" + created.ID
	}
	h.inertia.Redirect(w, r.WithContext(ctx), redirect, http.StatusSeeOther)
}

// Show handles GET /webhooks/{id} — detail + edit form.
func (h *WebhooksHandler) Show(w http.ResponseWriter, r *http.Request, id string) {
	props := inertia.Props{
		"site":       meta.ForRequest(h.site, r),
		"webhook":    nil,
		"triggers":   kiwify.WebhookTriggers,
		"defaultUrl": h.defaultReceiveURL(),
		"errors":     map[string]string{},
	}
	if msg, ok := flash.MessageFromRequest(r); ok {
		props["flash"] = inertia.Flash{msg.Kind: msg.Message}
	}

	api, err := h.webhooksAPI()
	if err != nil {
		props["errors"] = map[string]string{"client": "Credenciais Kiwify não configuradas ou inválidas."}
		_ = h.inertia.Render(w, r, "WebhookForm", props)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	wh, err := api.GetWebhook(ctx, id)
	if err != nil {
		props["errors"] = map[string]string{"webhook": userFacingAPIError(err, "Não foi possível carregar o webhook.")}
		_ = h.inertia.Render(w, r, "WebhookForm", props)
		return
	}
	props["webhook"] = wh
	_ = h.inertia.Render(w, r, "WebhookForm", props)
}

// Update handles POST /webhooks/{id}. Audits webhooks.update.
func (h *WebhooksHandler) Update(w http.ResponseWriter, r *http.Request, id string) {
	if err := httpx.ParseFormOrJSON(r); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	in, ve := parseWebhookForm(r)
	if len(ve) > 0 {
		msg := "Preencha os campos obrigatórios do webhook."
		for _, v := range ve {
			if s, ok := v.(string); ok && s != "" {
				msg = s
				break
			}
		}
		ctx := inertia.SetFlash(r.Context(), inertia.Flash{"error": msg})
		h.inertia.Redirect(w, r.WithContext(ctx), "/webhooks/"+id, http.StatusSeeOther)
		return
	}

	userID, _ := session.UserID(r)
	ip := middleware.ClientIP(r, h.cfg)
	summary, _ := json.Marshal(map[string]any{
		"id":       id,
		"name":     in.Name,
		"url":      in.URL,
		"products": in.Products,
		"triggers": in.Triggers,
	})

	respStatus := http.StatusOK
	respBody := `{"ok":true}`
	var updateErr error

	api, err := h.webhooksAPI()
	if err != nil {
		updateErr = err
		respStatus = http.StatusInternalServerError
		respBody = truncateAuditBody(err.Error())
	} else {
		ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
		defer cancel()
		updated, err := api.UpdateWebhook(ctx, id, in)
		updateErr = err
		if updateErr != nil {
			if apiErr, ok := kiwify.AsAPIError(updateErr); ok {
				respStatus = apiErr.Status
				if apiErr.Body != "" {
					respBody = truncateAuditBody(apiErr.Body)
				} else {
					respBody = truncateAuditBody(apiErr.Error())
				}
			} else {
				respStatus = http.StatusInternalServerError
				respBody = truncateAuditBody(updateErr.Error())
			}
		} else if updated.ID != "" {
			raw, _ := json.Marshal(map[string]any{"ok": true, "id": updated.ID})
			respBody = string(raw)
		}
	}

	_, _ = h.store.InsertAuditLog(store.AuditLog{
		UserID:         userID,
		Action:         "webhooks.update",
		ResourceType:   "webhook",
		ResourceID:     id,
		RequestSummary: string(summary),
		ResponseStatus: respStatus,
		ResponseBody:   respBody,
		IP:             ip,
	})

	flashKind := "success"
	flashMsg := "Webhook atualizado com sucesso."
	if updateErr != nil {
		flashKind = "error"
		flashMsg = userFacingAPIError(updateErr, "Não foi possível atualizar o webhook.")
	}
	ctx := inertia.SetFlash(r.Context(), inertia.Flash{flashKind: flashMsg})
	h.inertia.Redirect(w, r.WithContext(ctx), "/webhooks/"+id, http.StatusSeeOther)
}

// Delete handles POST /webhooks/{id}/delete. Audits webhooks.delete.
func (h *WebhooksHandler) Delete(w http.ResponseWriter, r *http.Request, id string) {
	userID, _ := session.UserID(r)
	ip := middleware.ClientIP(r, h.cfg)
	summary, _ := json.Marshal(map[string]any{"id": id})

	respStatus := http.StatusOK
	respBody := `{"ok":true}`
	var deleteErr error

	api, err := h.webhooksAPI()
	if err != nil {
		deleteErr = err
		respStatus = http.StatusInternalServerError
		respBody = truncateAuditBody(err.Error())
	} else {
		ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
		defer cancel()
		deleteErr = api.DeleteWebhook(ctx, id)
		if deleteErr != nil {
			if apiErr, ok := kiwify.AsAPIError(deleteErr); ok {
				respStatus = apiErr.Status
				if apiErr.Body != "" {
					respBody = truncateAuditBody(apiErr.Body)
				} else {
					respBody = truncateAuditBody(apiErr.Error())
				}
			} else {
				respStatus = http.StatusInternalServerError
				respBody = truncateAuditBody(deleteErr.Error())
			}
		}
	}

	_, _ = h.store.InsertAuditLog(store.AuditLog{
		UserID:         userID,
		Action:         "webhooks.delete",
		ResourceType:   "webhook",
		ResourceID:     id,
		RequestSummary: string(summary),
		ResponseStatus: respStatus,
		ResponseBody:   respBody,
		IP:             ip,
	})

	if deleteErr != nil {
		ctx := inertia.SetFlash(r.Context(), inertia.Flash{
			"error": userFacingAPIError(deleteErr, "Não foi possível excluir o webhook."),
		})
		h.inertia.Redirect(w, r.WithContext(ctx), "/webhooks/"+id, http.StatusSeeOther)
		return
	}

	ctx := inertia.SetFlash(r.Context(), inertia.Flash{"success": "Webhook excluído com sucesso."})
	h.inertia.Redirect(w, r.WithContext(ctx), "/webhooks", http.StatusSeeOther)
}

func parseWebhookForm(r *http.Request) (kiwify.WebhookInput, inertia.ValidationErrors) {
	in := kiwify.WebhookInput{
		Name:     strings.TrimSpace(r.FormValue("name")),
		URL:      strings.TrimSpace(r.FormValue("url")),
		Products: strings.TrimSpace(r.FormValue("products")),
		Token:    strings.TrimSpace(r.FormValue("token")),
	}
	// triggers may be multi-value checkboxes or a comma-separated string.
	if vals := r.Form["triggers"]; len(vals) > 0 {
		var cleaned []string
		for _, v := range vals {
			for _, part := range strings.Split(v, ",") {
				part = strings.TrimSpace(part)
				if part != "" {
					cleaned = append(cleaned, part)
				}
			}
		}
		in.Triggers = cleaned
	}
	if in.Triggers == nil {
		in.Triggers = []string{}
	}
	if in.Products == "" {
		in.Products = "all"
	}

	ve := inertia.ValidationErrors{}
	if in.Name == "" {
		ve["name"] = "Nome é obrigatório."
	}
	if in.URL == "" {
		ve["url"] = "URL é obrigatória."
	}
	if len(in.Triggers) == 0 {
		ve["triggers"] = "Selecione ao menos um gatilho."
	}
	return in, ve
}
