package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/puppe1990/cais/pkg/cais"
	"github.com/puppe1990/cais/pkg/cais/session"

	"github.com/puppe1990/kiwify_dashboard/internal/kiwify"
	"github.com/puppe1990/kiwify_dashboard/internal/store"
)

type fakeWebhooksAPI struct {
	listFn   func(ctx context.Context, q kiwify.WebhooksQuery) (kiwify.WebhooksPage, error)
	getFn    func(ctx context.Context, id string) (kiwify.Webhook, error)
	createFn func(ctx context.Context, in kiwify.WebhookInput) (kiwify.Webhook, error)
	updateFn func(ctx context.Context, id string, in kiwify.WebhookInput) (kiwify.Webhook, error)
	deleteFn func(ctx context.Context, id string) error

	lastQuery kiwify.WebhooksQuery
	createIn  kiwify.WebhookInput
	updateID  string
	updateIn  kiwify.WebhookInput
	deleteID  string
	listN     int
	getN      int
	createN   int
	updateN   int
	deleteN   int
}

func (f *fakeWebhooksAPI) ListWebhooks(ctx context.Context, q kiwify.WebhooksQuery) (kiwify.WebhooksPage, error) {
	f.listN++
	f.lastQuery = q
	if f.listFn != nil {
		return f.listFn(ctx, q)
	}
	return kiwify.WebhooksPage{Data: []kiwify.Webhook{}}, nil
}

func (f *fakeWebhooksAPI) GetWebhook(ctx context.Context, id string) (kiwify.Webhook, error) {
	f.getN++
	if f.getFn != nil {
		return f.getFn(ctx, id)
	}
	return kiwify.Webhook{ID: id, Name: "wh", URL: "https://x", Products: "all", Triggers: []string{"compra_aprovada"}}, nil
}

func (f *fakeWebhooksAPI) CreateWebhook(ctx context.Context, in kiwify.WebhookInput) (kiwify.Webhook, error) {
	f.createN++
	f.createIn = in
	if f.createFn != nil {
		return f.createFn(ctx, in)
	}
	return kiwify.Webhook{ID: "wh-new", Name: in.Name, URL: in.URL, Products: in.Products, Triggers: in.Triggers}, nil
}

func (f *fakeWebhooksAPI) UpdateWebhook(ctx context.Context, id string, in kiwify.WebhookInput) (kiwify.Webhook, error) {
	f.updateN++
	f.updateID = id
	f.updateIn = in
	if f.updateFn != nil {
		return f.updateFn(ctx, id, in)
	}
	return kiwify.Webhook{ID: id, Name: in.Name, URL: in.URL, Products: in.Products, Triggers: in.Triggers}, nil
}

func (f *fakeWebhooksAPI) DeleteWebhook(ctx context.Context, id string) error {
	f.deleteN++
	f.deleteID = id
	if f.deleteFn != nil {
		return f.deleteFn(ctx, id)
	}
	return nil
}

func newWebhooksHandler(t *testing.T, api WebhooksAPI) (*WebhooksHandler, store.Store) {
	t.Helper()
	s := setupTestStore(t)
	h := NewWebhooksHandler(s, testAppSecret(), testSite(), cais.Config{AppURL: "https://cais.example.com"}, setupTestInertia(t), api)
	return h, s
}

func TestWebhooksList_Renders(t *testing.T) {
	fake := &fakeWebhooksAPI{
		listFn: func(ctx context.Context, q kiwify.WebhooksQuery) (kiwify.WebhooksPage, error) {
			return kiwify.WebhooksPage{
				Data: []kiwify.Webhook{
					{ID: "wh-1", Name: "ops", URL: "https://ex/hook", Products: "all", Triggers: []string{"compra_aprovada"}},
				},
				Pagination: kiwify.Pagination{Count: 1, PageNumber: 1, PageSize: 50},
			}, nil
		},
	}
	h, _ := newWebhooksHandler(t, fake)

	req := inertiaRequest(http.MethodGet, "/webhooks", nil)
	rr := httptest.NewRecorder()
	h.List(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rr.Code, rr.Body.String())
	}
	assertInertiaComponent(t, rr, "Webhooks")
	list := assertInertiaProp(t, rr, "webhooks").([]any)
	if len(list) != 1 {
		t.Fatalf("webhooks len = %d", len(list))
	}
	wh := list[0].(map[string]any)
	if wh["id"] != "wh-1" || wh["name"] != "ops" {
		t.Fatalf("webhook = %v", wh)
	}
	if fake.lastQuery.PageSize != "50" {
		t.Fatalf("page size = %q", fake.lastQuery.PageSize)
	}
}

func TestWebhooksNew_PrefillsDefaultURL(t *testing.T) {
	h, s := newWebhooksHandler(t, &fakeWebhooksAPI{})
	if err := s.SaveKiwifySettings(store.KiwifySettings{
		AccountID: "a", ClientID: "c", ClientSecretCiphertext: "x",
		WebhookReceiveToken: "recv-token-hex",
	}); err != nil {
		t.Fatal(err)
	}

	req := inertiaRequest(http.MethodGet, "/webhooks/new", nil)
	rr := httptest.NewRecorder()
	h.New(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d", rr.Code)
	}
	assertInertiaComponent(t, rr, "WebhookForm")
	want := "https://cais.example.com/webhooks/kiwify/recv-token-hex"
	if v := assertInertiaProp(t, rr, "defaultUrl"); v != want {
		t.Fatalf("defaultUrl = %v, want %s", v, want)
	}
}

func TestWebhooksCreate_SuccessWritesAudit(t *testing.T) {
	fake := &fakeWebhooksAPI{}
	h, s := newWebhooksHandler(t, fake)

	form := url.Values{
		"name":     {"meu webhook"},
		"url":      {"https://cais.example.com/webhooks/kiwify/tok"},
		"products": {"all"},
		"triggers": {"compra_aprovada", "compra_reembolsada"},
	}
	req := inertiaRequest(http.MethodPost, "/webhooks", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = session.WithUserID(req, 7)
	req.RemoteAddr = "203.0.113.5:1"
	rr := httptest.NewRecorder()
	h.Create(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("status = %d body=%s", rr.Code, rr.Body.String())
	}
	if loc := rr.Header().Get("Location"); loc != "/webhooks/wh-new" {
		t.Fatalf("Location = %q", loc)
	}
	if fake.createN != 1 {
		t.Fatalf("create calls = %d", fake.createN)
	}
	if fake.createIn.Name != "meu webhook" || fake.createIn.Products != "all" {
		t.Fatalf("create in = %+v", fake.createIn)
	}
	if len(fake.createIn.Triggers) != 2 {
		t.Fatalf("triggers = %v", fake.createIn.Triggers)
	}

	logs, err := s.ListAuditLogs(10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(logs) != 1 || logs[0].Action != "webhooks.create" {
		t.Fatalf("audit = %+v", logs)
	}
	if logs[0].ResourceID != "wh-new" || logs[0].UserID != 7 {
		t.Fatalf("audit fields = %+v", logs[0])
	}
}

func TestWebhooksCreate_ValidationMissingTriggers(t *testing.T) {
	fake := &fakeWebhooksAPI{}
	h, _ := newWebhooksHandler(t, fake)

	form := url.Values{
		"name":     {"x"},
		"url":      {"https://x"},
		"products": {"all"},
	}
	req := inertiaRequest(http.MethodPost, "/webhooks", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	h.Create(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d", rr.Code)
	}
	assertInertiaComponent(t, rr, "WebhookForm")
	if fake.createN != 0 {
		t.Fatal("should not call API on validation error")
	}
}

func TestWebhooksUpdate_SuccessWritesAudit(t *testing.T) {
	fake := &fakeWebhooksAPI{}
	h, s := newWebhooksHandler(t, fake)

	form := url.Values{
		"name":     {"updated"},
		"url":      {"https://ex/hook"},
		"products": {"prod-1"},
		"triggers": {"pix_gerado"},
	}
	req := inertiaRequest(http.MethodPost, "/webhooks/wh-1", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = session.WithUserID(req, 3)
	rr := httptest.NewRecorder()
	h.Update(rr, req, "wh-1")

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("status = %d", rr.Code)
	}
	if fake.updateN != 1 || fake.updateID != "wh-1" {
		t.Fatalf("update = %d id=%s", fake.updateN, fake.updateID)
	}
	logs, err := s.ListAuditLogs(10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(logs) != 1 || logs[0].Action != "webhooks.update" {
		t.Fatalf("audit = %+v", logs)
	}
}

func TestWebhooksDelete_SuccessWritesAudit(t *testing.T) {
	fake := &fakeWebhooksAPI{}
	h, s := newWebhooksHandler(t, fake)

	req := inertiaRequest(http.MethodDelete, "/webhooks/wh-del", nil)
	req = session.WithUserID(req, 9)
	rr := httptest.NewRecorder()
	h.Delete(rr, req, "wh-del")

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("status = %d", rr.Code)
	}
	if loc := rr.Header().Get("Location"); loc != "/webhooks" {
		t.Fatalf("Location = %q", loc)
	}
	if fake.deleteID != "wh-del" {
		t.Fatalf("delete id = %q", fake.deleteID)
	}
	logs, err := s.ListAuditLogs(10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(logs) != 1 || logs[0].Action != "webhooks.delete" {
		t.Fatalf("audit = %+v", logs)
	}
}

func TestWebhooksDelete_FailureStillAudits(t *testing.T) {
	fake := &fakeWebhooksAPI{
		deleteFn: func(ctx context.Context, id string) error {
			return errors.New("boom")
		},
	}
	h, s := newWebhooksHandler(t, fake)

	req := inertiaRequest(http.MethodDelete, "/webhooks/wh-x", nil)
	req = session.WithUserID(req, 1)
	rr := httptest.NewRecorder()
	h.Delete(rr, req, "wh-x")

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("status = %d", rr.Code)
	}
	if loc := rr.Header().Get("Location"); loc != "/webhooks/wh-x" {
		t.Fatalf("Location = %q", loc)
	}
	logs, err := s.ListAuditLogs(10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(logs) != 1 || logs[0].Action != "webhooks.delete" {
		t.Fatalf("audit = %+v", logs)
	}
	if logs[0].ResponseStatus == http.StatusOK {
		t.Fatalf("expected non-OK audit status, got %d", logs[0].ResponseStatus)
	}
}
