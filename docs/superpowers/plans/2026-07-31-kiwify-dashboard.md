# Kiwify Ops Dashboard Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a personal Kiwify operations dashboard on cais that live-proxies the full Public API (reads + supported mutations), stores encrypted credentials, audits sensitive actions, and receives webhooks.

**Architecture:** Scaffold with `cais new` (Go + Inertia + Svelte 5 + SQLite). Server-side `internal/kiwify` HTTP client with OAuth token cache. Local SQLite holds settings, audit logs, and inbound webhook events only. UI is Kiwi green (PT-BR).

**Tech Stack:** Go 1.26, cais (`github.com/puppe1990/cais`), gonertia/v3, Svelte 5, Tailwind 3, modernc.org/sqlite, Kiwify Public API `https://public-api.kiwify.com/v1`

**Spec:** `docs/superpowers/specs/2026-07-31-kiwify-dashboard-design.md`

---

## File map (target)

```
cmd/server/main.go
internal/
  app/app.go
  app/routes.go
  crypto/secret.go
  crypto/secret_test.go
  kiwify/
    client.go          # HTTP + GetToken + do()
    client_test.go
    errors.go
    types.go           # request/response DTOs (minimal fields used by UI)
    sales.go products.go finance.go affiliates.go webhooks.go account.go events.go stats.go
  store/
    store.go           # Store interface + SQLiteStore
    settings.go
    audit.go
    webhook_events.go
    migrations/003_kiwify_ops.sql
  handlers/
    setup.go settings.go dashboard.go
    sales.go products.go finance.go affiliates.go
    webhooks.go events.go audit.go account.go
    kiwify_webhook.go
    helpers.go         # flash, require settings, audit helper
  middleware/require_setup.go
web/src/
  components/AppLayout.svelte   # Kiwi green sidebar
  components/StatCard.svelte
  components/ConfirmModal.svelte
  components/DataTable.svelte
  components/FlashBanner.svelte
  pages/Setup.svelte Settings.svelte Dashboard.svelte
  pages/Sales.svelte SaleShow.svelte Products.svelte ProductShow.svelte
  pages/Finance.svelte Affiliates.svelte AffiliateShow.svelte
  pages/Webhooks.svelte WebhookForm.svelte Events.svelte
  pages/Audit.svelte Account.svelte
```

**Kiwify API paths (v1):**

| Method         | Path                                 | Notes                                                             |
| -------------- | ------------------------------------ | ----------------------------------------------------------------- |
| POST           | `/oauth/token`                       | form: `client_id`, `client_secret` → `access_token`, `expires_in` |
| GET            | `/sales`                             | query: `start_date`, `end_date` (max 90d), pagination             |
| GET            | `/sales/{id}`                        | detail                                                            |
| POST           | `/sales/{id}/refund`                 | optional body `pixKey`                                            |
| GET            | `/stats`                             | sales statistics                                                  |
| GET            | `/products`, `/products/{id}`        | read-only                                                         |
| GET            | `/balance`                           | balances                                                          |
| GET            | `/payouts`, `/payouts/{id}`          | list/detail                                                       |
| POST           | `/payouts/`                          | body `{ "amount": number }`                                       |
| GET/POST       | `/affiliates`, `/affiliates/{id}`    | list/get/edit (PUT/PATCH or POST per docs)                        |
| GET/POST       | `/webhooks`                          | CRUD                                                              |
| GET/PUT/DELETE | `/webhooks/{id}`                     | per docs                                                          |
| GET            | `/account` (or account-details path) | account                                                           |
| GET            | `/events` participants               | per docs                                                          |

Verify exact affiliate edit method and account path against docs when implementing; wrap in client methods so handlers stay stable.

Headers on authenticated calls:

```
Authorization: Bearer <token>
x-kiwify-account-id: <account_id>
```

---

### Task 1: Scaffold cais app into this repo

**Files:**

- Create: full cais tree under project root (preserve existing `docs/`, `.gitignore`, `.git`)
- Modify: `.gitignore` if scaffold adds entries worth merging

- [ ] **Step 1: Scaffold into a temp directory**

```bash
export PATH="$HOME/go/bin:$PATH"
cd /tmp
rm -rf kd-scaffold
cais new kiwify_dashboard /tmp/kd-scaffold --module github.com/puppe1990/kiwify_dashboard
```

Expected: app created with auth, Dashboard, Inertia, migrations 001–002.

- [ ] **Step 2: Copy scaffold into the project without wiping docs**

```bash
cd /Users/matheuspuppe/Desktop/Projetos/kiwify_dashboard
# copy everything except .git
rsync -a --exclude '.git' /tmp/kd-scaffold/ ./
# ensure design docs still present
test -f docs/superpowers/specs/2026-07-31-kiwify-dashboard-design.md
```

If rsync overwrites `.gitignore`, re-add:

```
.firecrawl/
.superpowers/
*.db
.env
.env.*
!.env.example
```

- [ ] **Step 3: Install and verify boot**

```bash
export PATH="$HOME/go/bin:$PATH"
cais install
cais db migrate
cais test
```

Expected: tests pass (scaffold defaults). Fix module/replace issues if any.

- [ ] **Step 4: Commit**

```bash
git add -A
git status
git commit -m "chore: scaffold cais app for Kiwify dashboard"
```

---

### Task 2: Crypto helper for secrets

**Files:**

- Create: `internal/crypto/secret.go`
- Create: `internal/crypto/secret_test.go`

- [ ] **Step 1: Write failing tests**

```go
package crypto_test

import (
	"testing"

	"github.com/puppe1990/kiwify_dashboard/internal/crypto"
)

func TestEncryptDecryptRoundTrip(t *testing.T) {
	key := crypto.DeriveKey("test-app-secret-for-unit-tests-32b")
	ct, err := crypto.Encrypt(key, "super-secret-value")
	if err != nil {
		t.Fatal(err)
	}
	if ct == "super-secret-value" {
		t.Fatal("ciphertext must not equal plaintext")
	}
	pt, err := crypto.Decrypt(key, ct)
	if err != nil {
		t.Fatal(err)
	}
	if pt != "super-secret-value" {
		t.Fatalf("got %q", pt)
	}
}

func TestDecryptWrongKeyFails(t *testing.T) {
	key1 := crypto.DeriveKey("secret-one-aaaaaaaaaaaaaaaaaaaa")
	key2 := crypto.DeriveKey("secret-two-bbbbbbbbbbbbbbbbbbbb")
	ct, err := crypto.Encrypt(key1, "x")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := crypto.Decrypt(key2, ct); err == nil {
		t.Fatal("expected error")
	}
}
```

- [ ] **Step 2: Run tests — expect FAIL**

```bash
go test ./internal/crypto/ -v
```

Expected: package not found / undefined.

- [ ] **Step 3: Implement AES-GCM helpers**

```go
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
)

// DeriveKey returns a 32-byte key from an arbitrary secret string.
func DeriveKey(appSecret string) []byte {
	sum := sha256.Sum256([]byte(appSecret))
	return sum[:]
}

func Encrypt(key []byte, plaintext string) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	out := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(out), nil
}

func Decrypt(key []byte, ciphertext string) (string, error) {
	raw, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(raw) < gcm.NonceSize() {
		return "", fmt.Errorf("ciphertext too short")
	}
	nonce, ct := raw[:gcm.NonceSize()], raw[gcm.NonceSize():]
	pt, err := gcm.Open(nil, nonce, ct, nil)
	if err != nil {
		return "", err
	}
	return string(pt), nil
}
```

- [ ] **Step 4: Run tests — expect PASS**

```bash
go test ./internal/crypto/ -v
```

- [ ] **Step 5: Commit**

```bash
git add internal/crypto/
git commit -m "feat: add AES-GCM secret encrypt/decrypt helpers"
```

---

### Task 3: Migrations + store for settings, audit, webhook events

**Files:**

- Create: `internal/store/migrations/003_kiwify_ops.sql`
- Create: `internal/store/settings.go`
- Create: `internal/store/audit.go`
- Create: `internal/store/webhook_events.go`
- Create: `internal/store/settings_test.go` (or extend `store_test.go`)
- Modify: `internal/store/store.go` (extend `Store` interface)

- [ ] **Step 1: Add migration SQL**

```sql
-- internal/store/migrations/003_kiwify_ops.sql
CREATE TABLE IF NOT EXISTS kiwify_settings (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    account_id TEXT NOT NULL DEFAULT '',
    client_id TEXT NOT NULL DEFAULT '',
    client_secret_ciphertext TEXT NOT NULL DEFAULT '',
    oauth_access_token_ciphertext TEXT NOT NULL DEFAULT '',
    token_expires_at TEXT,
    webhook_receive_token TEXT NOT NULL DEFAULT '',
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

INSERT OR IGNORE INTO kiwify_settings (id) VALUES (1);

CREATE TABLE IF NOT EXISTS audit_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    action TEXT NOT NULL,
    resource_type TEXT NOT NULL DEFAULT '',
    resource_id TEXT NOT NULL DEFAULT '',
    request_summary TEXT NOT NULL DEFAULT '',
    response_status INTEGER NOT NULL DEFAULT 0,
    response_body TEXT NOT NULL DEFAULT '',
    ip TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS webhook_events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    event_type TEXT NOT NULL DEFAULT '',
    payload_json TEXT NOT NULL,
    headers_json TEXT NOT NULL DEFAULT '',
    received_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    processed_ok INTEGER NOT NULL DEFAULT 1
);

CREATE INDEX IF NOT EXISTS idx_audit_logs_created ON audit_logs(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_webhook_events_received ON webhook_events(received_at DESC);
```

- [ ] **Step 2: Write store tests (settings + audit + events)**

```go
func TestKiwifySettingsRoundTrip(t *testing.T) {
	s := newTestStore(t) // use existing test helper from scaffold
	err := s.SaveKiwifySettings(store.KiwifySettings{
		AccountID:              "acc1",
		ClientID:               "cid",
		ClientSecretCiphertext: "enc-secret",
		WebhookReceiveToken:    "tok123",
	})
	if err != nil {
		t.Fatal(err)
	}
	got, err := s.GetKiwifySettings()
	if err != nil {
		t.Fatal(err)
	}
	if got.AccountID != "acc1" || got.ClientID != "cid" || got.WebhookReceiveToken != "tok123" {
		t.Fatalf("%+v", got)
	}
}

func TestInsertAuditAndList(t *testing.T) {
	s := newTestStore(t)
	id, err := s.InsertAuditLog(store.AuditLog{
		UserID: 1, Action: "sales.refund", ResourceType: "sale", ResourceID: "ord1",
		RequestSummary: `{"id":"ord1"}`, ResponseStatus: 200, ResponseBody: `{"refunded":true}`, IP: "127.0.0.1",
	})
	if err != nil || id == 0 {
		t.Fatalf("id=%d err=%v", id, err)
	}
	logs, err := s.ListAuditLogs(50, 0)
	if err != nil || len(logs) != 1 {
		t.Fatalf("len=%d err=%v", len(logs), err)
	}
}

func TestInsertWebhookEvent(t *testing.T) {
	s := newTestStore(t)
	id, err := s.InsertWebhookEvent(store.WebhookEvent{
		EventType: "compra_aprovada", PayloadJSON: `{"a":1}`, HeadersJSON: `{}`, ProcessedOK: true,
	})
	if err != nil || id == 0 {
		t.Fatal(err)
	}
}
```

- [ ] **Step 3: Run tests — expect FAIL** (methods missing)

```bash
go test ./internal/store/ -run 'Kiwify|Audit|Webhook' -v
```

- [ ] **Step 4: Implement models + store methods**

Add types and methods:

```go
type KiwifySettings struct {
	AccountID                  string
	ClientID                   string
	ClientSecretCiphertext     string
	OAuthAccessTokenCiphertext string
	TokenExpiresAt             *time.Time
	WebhookReceiveToken        string
	UpdatedAt                  time.Time
}

type AuditLog struct {
	ID, UserID         int64
	Action             string
	ResourceType       string
	ResourceID         string
	RequestSummary     string
	ResponseStatus     int
	ResponseBody       string
	IP                 string
	CreatedAt          time.Time
}

type WebhookEvent struct {
	ID           int64
	EventType    string
	PayloadJSON  string
	HeadersJSON  string
	ReceivedAt   time.Time
	ProcessedOK  bool
}
```

Implement on `SQLiteStore` and extend `Store` interface:

- `GetKiwifySettings() (KiwifySettings, error)`
- `SaveKiwifySettings(KiwifySettings) error`
- `UpdateOAuthToken(ciphertext string, expiresAt time.Time) error`
- `InsertAuditLog(AuditLog) (int64, error)`
- `ListAuditLogs(limit, offset int) ([]AuditLog, error)`
- `InsertWebhookEvent(WebhookEvent) (int64, error)`
- `ListWebhookEvents(limit, offset int) ([]WebhookEvent, error)`
- `Configured() (bool, error)` — true when account_id, client_id, secret ciphertext non-empty

`SaveKiwifySettings` should generate `WebhookReceiveToken` with `crypto/rand` hex if empty.

- [ ] **Step 5: Run tests — expect PASS**

```bash
go test ./internal/store/ -v
cais db migrate   # or boot app so migrations apply
```

- [ ] **Step 6: Commit**

```bash
git add internal/store/
git commit -m "feat: add settings, audit, and webhook_events store"
```

---

### Task 4: Kiwify HTTP client (token + do + errors)

**Files:**

- Create: `internal/kiwify/errors.go`
- Create: `internal/kiwify/client.go`
- Create: `internal/kiwify/client_test.go`
- Create: `internal/kiwify/types.go` (minimal shared types)

- [ ] **Step 1: Write failing client tests with httptest**

```go
package kiwify_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/puppe1990/kiwify_dashboard/internal/kiwify"
)

type memTokenStore struct {
	token string
	exp   time.Time
}

func (m *memTokenStore) LoadToken() (string, time.Time, error) { return m.token, m.exp, nil }
func (m *memTokenStore) SaveToken(token string, exp time.Time) error {
	m.token, m.exp = token, exp
	return nil
}

func TestGetTokenCachesUntilExpiry(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/oauth/token" {
			calls.Add(1)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": "tok-abc",
				"token_type":   "Bearer",
				"expires_in":   "3600",
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	ts := &memTokenStore{}
	c := kiwify.NewClient(kiwify.Config{
		BaseURL:      srv.URL + "/v1",
		AccountID:    "acc",
		ClientID:     "cid",
		ClientSecret: "sec",
		TokenStore:   ts,
		HTTPClient:   srv.Client(),
	})
	tok1, err := c.GetToken(context.Background())
	if err != nil || tok1 != "tok-abc" {
		t.Fatalf("%v %q", err, tok1)
	}
	tok2, err := c.GetToken(context.Background())
	if err != nil || tok2 != "tok-abc" {
		t.Fatal(err)
	}
	if calls.Load() != 1 {
		t.Fatalf("oauth called %d times", calls.Load())
	}
}

func TestDoSendsAuthHeaders(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/oauth/token" {
			_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "t", "expires_in": "3600"})
			return
		}
		if r.Header.Get("Authorization") != "Bearer t" {
			t.Errorf("auth %q", r.Header.Get("Authorization"))
		}
		if r.Header.Get("x-kiwify-account-id") != "acc" {
			t.Errorf("account %q", r.Header.Get("x-kiwify-account-id"))
		}
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	c := kiwify.NewClient(kiwify.Config{
		BaseURL: srv.URL + "/v1", AccountID: "acc", ClientID: "c", ClientSecret: "s",
		TokenStore: &memTokenStore{}, HTTPClient: srv.Client(),
	})
	var out map[string]any
	if err := c.GetJSON(context.Background(), "/products", nil, &out); err != nil {
		t.Fatal(err)
	}
}

func TestAPIError429(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/oauth/token" {
			_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "t", "expires_in": "3600"})
			return
		}
		w.WriteHeader(429)
		_, _ = w.Write([]byte(`{"message":"rate limited"}`))
	}))
	defer srv.Close()
	c := kiwify.NewClient(kiwify.Config{
		BaseURL: srv.URL + "/v1", AccountID: "a", ClientID: "c", ClientSecret: "s",
		TokenStore: &memTokenStore{}, HTTPClient: srv.Client(),
	})
	err := c.GetJSON(context.Background(), "/sales", map[string]string{"start_date": "x", "end_date": "y"}, nil)
	apiErr, ok := kiwify.AsAPIError(err)
	if !ok || apiErr.Status != 429 {
		t.Fatalf("%v", err)
	}
	if apiErr.UserMessage == "" {
		t.Fatal("expected PT-BR user message")
	}
}
```

- [ ] **Step 2: Run — expect FAIL**

```bash
go test ./internal/kiwify/ -v
```

- [ ] **Step 3: Implement client core**

`TokenStore` interface:

```go
type TokenStore interface {
	LoadToken() (token string, expiresAt time.Time, error)
	SaveToken(token string, expiresAt time.Time) error
}
```

`Client` responsibilities:

- `GetToken(ctx)`: if token valid for >60s, return it; else POST form-urlencoded `client_id`/`client_secret` to `{BaseURL}/oauth/token`, parse `access_token` + `expires_in` (string or number), `SaveToken`, return token.
- `do(ctx, method, path, query, body, out)`: attach headers; on 401 once, clear token, refresh, retry once.
- `GetJSON`, `PostJSON`, `PutJSON`, `DeleteJSON` wrappers.
- `APIError{Status int, Code string, Message string, UserMessage string, Body string}` with `AsAPIError`.

User messages (PT-BR examples):

- 401: "Credenciais Kiwify inválidas ou token expirado. Verifique Configurações."
- 429: "Limite da API Kiwify atingido (100/min). Aguarde um minuto e tente de novo."
- 400: use API `message` if present, else "Requisição inválida."
- 5xx: "Erro no servidor da Kiwify. Tente mais tarde."

Default `BaseURL`: `https://public-api.kiwify.com/v1`

Adapter that implements `TokenStore` using `store` + `crypto` will be wired in app deps (Task 5) — keep client free of SQLite imports.

- [ ] **Step 4: Run — expect PASS**

```bash
go test ./internal/kiwify/ -v
```

- [ ] **Step 5: Commit**

```bash
git add internal/kiwify/
git commit -m "feat: add Kiwify API client with OAuth token cache"
```

---

### Task 5: Wire deps — client factory, APP_SECRET, require-setup middleware

**Files:**

- Create: `internal/kiwify/token_store_adapter.go` (or `internal/app/kiwify_factory.go`)
- Create: `internal/middleware/require_setup.go`
- Create: `internal/middleware/require_setup_test.go`
- Modify: `internal/app/app.go` (Deps)
- Modify: `cmd/server` / config loading for `APP_SECRET`
- Create: `.env.example`

- [ ] **Step 1: Token store adapter**

```go
// Decrypts token ciphertext on load; encrypts on save.
type DBTokenStore struct {
	Store store.Store
	Key   []byte
}

func (t *DBTokenStore) LoadToken() (string, time.Time, error) {
	s, err := t.Store.GetKiwifySettings()
	// decrypt OAuthAccessTokenCiphertext; return expires
}

func (t *DBTokenStore) SaveToken(token string, exp time.Time) error {
	// encrypt + Store.UpdateOAuthToken
}
```

`NewKiwifyClientFromStore(st store.Store, key []byte, httpClient *http.Client) (*kiwify.Client, error)`:

- load settings, decrypt secret, build Config.

- [ ] **Step 2: RequireSetup middleware**

If `!Configured()`, redirect to `/setup` (skip paths: `/login`, `/signup`, `/setup`, `/webhooks/kiwify/`).

Test with httptest: unconfigured → 303 `/setup`; configured → next handler.

- [ ] **Step 3: APP_SECRET**

Read from env; if empty in development, log warning and use a stable dev default **only when `ENV=development`** (document in README). Production must set `APP_SECRET`.

`.env.example`:

```
APP_SECRET=change-me-to-a-long-random-string
```

- [ ] **Step 4: Wire Deps**

```go
type Deps struct {
	// existing fields...
	AppSecret []byte
}
```

Handlers receive `store` + secret and build client per-request or cache client after settings load.

- [ ] **Step 5: Tests + commit**

```bash
go test ./internal/middleware/ ./internal/kiwify/ ./internal/app/ -count=1
git add internal/middleware internal/app internal/kiwify .env.example
git commit -m "feat: wire Kiwify client factory and require-setup middleware"
```

---

### Task 6: Setup + Settings pages

**Files:**

- Create: `internal/handlers/setup.go`, `setup_test.go`
- Create: `internal/handlers/settings.go`, `settings_test.go`
- Create: `web/src/pages/Setup.svelte`, `Settings.svelte`
- Modify: `internal/app/routes.go`

- [ ] **Step 1: Handler tests**

- `GET /setup` authenticated → renders Setup (or 200 inertia)
- `POST /setup` with client_id, client_secret, account_id → encrypts secret, saves settings, generates webhook token, redirects `/`
- Settings GET never includes plaintext secret (prop `hasSecret: true`, masked placeholder)
- POST settings updates fields; empty secret keeps previous

- [ ] **Step 2: Implement handlers**

Use `httpx.ParseFormOrJSON`, validate non-empty fields, `crypto.Encrypt`, `SaveKiwifySettings`.

Optional: after save, try `GetToken` and flash success/failure without blocking save permanently (or block if OAuth fails — prefer **block save only on encrypt/db errors**; flash warning if OAuth test fails).

Settings props:

```go
inertia.Props{
  "accountId": settings.AccountID,
  "clientId": settings.ClientID,
  "hasSecret": settings.ClientSecretCiphertext != "",
  "webhookReceiveURL": absoluteURL + "/webhooks/kiwify/" + settings.WebhookReceiveToken,
  "site": meta.ForRequest(...),
}
```

- [ ] **Step 3: Svelte pages**

`Setup.svelte` / `Settings.svelte`: form with `useForm({ client_id, client_secret, account_id })`, Kiwi green styling basics, PT-BR labels.

- [ ] **Step 4: Routes**

```go
r.Group(middleware.RequireAuth("/login"), func(g *cais.Router) {
  g.Get("/setup", setup.Get)
  g.Post("/setup", setup.Post)
  g.Group(requireSetup, func(g2 *cais.Router) {
    g2.Get("/settings", settings.Get)
    g2.Post("/settings", settings.Post)
    // other app routes later
  })
})
```

- [ ] **Step 5: Manual smoke + commit**

```bash
cais test
git add internal/handlers web/src/pages internal/app/routes.go
git commit -m "feat: setup and settings for Kiwify API credentials"
```

---

### Task 7: App layout (Kiwi green) + shared components

**Files:**

- Modify: `web/src/components/AppLayout.svelte`
- Create: `web/src/components/StatCard.svelte`, `ConfirmModal.svelte`, `DataTable.svelte`, `FlashBanner.svelte`
- Modify: Tailwind config if needed for green palette

- [ ] **Step 1: Restyle AppLayout**

Sidebar `#14532d`, active `#166534`, accent `#22c55e`, main bg `#f0fdf4`.
Nav links: Dashboard, Vendas, Produtos, Financeiro, Afiliados, Webhooks, Eventos, Auditoria, Conta, Configurações, Sair.

- [ ] **Step 2: Shared components**

- `StatCard`: title, value, optional hint
- `ConfirmModal`: title, body slots, confirm/cancel; emits confirm
- `DataTable`: headers + rows slots or props
- `FlashBanner`: reads Inertia flash success/error

- [ ] **Step 3: Visual check with `cais dev`** — login seed `demo@example.com` / `password`

- [ ] **Step 4: Commit**

```bash
git add web/src
git commit -m "feat: Kiwi green layout and shared UI components"
```

---

### Task 8: Domain client methods + Dashboard

**Files:**

- Create: `internal/kiwify/sales.go`, `stats.go`, `finance.go`, `products.go` (+ tests with httptest fixtures)
- Create: `internal/handlers/dashboard.go` (replace scaffold)
- Modify: `web/src/pages/Dashboard.svelte`

- [ ] **Step 1: Client methods (TDD per method group)**

```go
func (c *Client) ListSales(ctx context.Context, q SalesQuery) (SalesPage, error)
func (c *Client) GetSale(ctx context.Context, id string) (Sale, error)
func (c *Client) SalesStats(ctx context.Context, q StatsQuery) (Stats, error)
func (c *Client) ListBalances(ctx context.Context) (Balances, error)
func (c *Client) ListProducts(ctx context.Context, q PageQuery) (ProductsPage, error)
```

Use `json.RawMessage` or maps only if types are unstable; prefer typed fields used by UI (`id`, `reference`, amounts, status, product name, dates).

- [ ] **Step 2: Dashboard handler**

Fetch in parallel (errgroup): stats (last 30d), balances, sales list page 1 (last 30d), local webhook events (limit 10).

On partial failure, still render with error flash for failed section.

- [ ] **Step 3: Dashboard.svelte** — 4 StatCards + last sales table + events feed

- [ ] **Step 4: Tests + commit**

```bash
go test ./internal/kiwify/ ./internal/handlers/ -count=1
git add internal/kiwify internal/handlers web/src/pages/Dashboard.svelte
git commit -m "feat: dashboard with live sales stats, balance, and events"
```

---

### Task 9: Sales list, detail, refund (+ audit)

**Files:**

- Create/extend: `internal/kiwify/sales.go` (`RefundSale`)
- Create: `internal/handlers/sales.go`, `sales_test.go`
- Create: `web/src/pages/Sales.svelte`, `SaleShow.svelte`
- Modify: `routes.go`

- [ ] **Step 1: Client refund test**

httptest expects `POST /v1/sales/{id}/refund` → `{"refunded":true}`.

- [ ] **Step 2: Handler tests**

- List with default last 30 days; clamp range to 90 days max
- Refund POST without auth → redirect login
- Refund with auth → calls client, writes audit, flash, 303 to show page
- Refund API error → audit with status, flash error

Use interface:

```go
type SalesAPI interface {
	ListSales(ctx context.Context, q kiwify.SalesQuery) (kiwify.SalesPage, error)
	GetSale(ctx context.Context, id string) (kiwify.Sale, error)
	RefundSale(ctx context.Context, id string, pixKey string) error
}
```

Inject fake in tests.

- [ ] **Step 3: Implement handlers + ConfirmModal on SaleShow**

Audit action: `sales.refund`.

- [ ] **Step 4: Commit**

```bash
git add internal/handlers/sales.go internal/handlers/sales_test.go internal/kiwify web/src/pages
git commit -m "feat: sales list, detail, and refund with audit log"
```

---

### Task 10: Products (read-only)

**Files:**

- Create: `internal/handlers/products.go`, `products_test.go`
- Create: `web/src/pages/Products.svelte`, `ProductShow.svelte`
- Extend: `internal/kiwify/products.go`

- [ ] **Step 1: List + show handlers** calling client; 404-friendly errors
- [ ] **Step 2: Pages with table + detail; note “criar produto só no dashboard Kiwify”**
- [ ] **Step 3: Commit**

```bash
git commit -m "feat: read-only products list and detail"
```

---

### Task 11: Finance (balances, payouts, request payout)

**Files:**

- Create: `internal/kiwify/finance.go`, tests
- Create: `internal/handlers/finance.go`, `finance_test.go`
- Create: `web/src/pages/Finance.svelte`

- [ ] **Step 1: Client** `ListBalances`, `ListPayouts`, `CreatePayout(amount float64)`
- [ ] **Step 2: Handler** GET aggregates balances + payouts; POST payout with confirm + audit `finance.payout`
- [ ] **Step 3: UI** amount input + ConfirmModal (“Solicitar saque de R$ X”)
- [ ] **Step 4: Commit**

```bash
git commit -m "feat: finance balances, payouts, and payout request"
```

---

### Task 12: Affiliates (list, detail, edit)

**Files:**

- Create: `internal/kiwify/affiliates.go`, tests
- Create: `internal/handlers/affiliates.go`, tests
- Create: `web/src/pages/Affiliates.svelte`, `AffiliateShow.svelte`

- [ ] **Step 1: Confirm HTTP method/path from docs at implement time** (`GET /affiliates`, `GET /affiliates/{id}`, edit endpoint)
- [ ] **Step 2: Edit with confirm + audit `affiliates.edit`**
- [ ] **Step 3: Commit**

```bash
git commit -m "feat: affiliates list, detail, and edit"
```

---

### Task 13: Webhooks CRUD + public receiver + Events page

**Files:**

- Create: `internal/kiwify/webhooks.go`, tests
- Create: `internal/handlers/webhooks.go`, `kiwify_webhook.go`, `events.go` + tests
- Create: `web/src/pages/Webhooks.svelte`, `WebhookForm.svelte`, `Events.svelte`
- Modify: `routes.go` — public `POST /webhooks/kiwify/{token}` outside auth group

- [ ] **Step 1: Client CRUD tests** against httptest (`GET/POST /webhooks`, `GET/PUT/DELETE /webhooks/{id}`)

Create body fields from docs: `name`, `url`, `products` (`all` or id), `triggers` [], optional `token`.

- [ ] **Step 2: Receiver tests**

```go
// valid token → 200 + row in webhook_events
// wrong token → 404
// invalid JSON → 400
```

Infer `event_type` from payload keys commonly used by Kiwify (`order_status`, `webhook_event_type`, or top-level trigger name); if unknown store `unknown`.

- [ ] **Step 3: CRUD handlers with audit on create/update/delete**

Settings already shows receive URL — Webhooks form prefill default URL to that value when creating.

- [ ] **Step 4: Events page** lists `ListWebhookEvents` with payload expandable JSON

- [ ] **Step 5: Commit**

```bash
git commit -m "feat: webhooks CRUD, secure receiver, and events feed"
```

---

### Task 14: Account, Audit pages, Account API

**Files:**

- Create: `internal/kiwify/account.go`, `events_api.go` (participants if needed)
- Create: `internal/handlers/account.go`, `audit.go` + tests
- Create: `web/src/pages/Account.svelte`, `Audit.svelte`

- [ ] **Step 1: Account page** — live API account details
- [ ] **Step 2: Audit page** — table from `ListAuditLogs` (filter optional later)
- [ ] **Step 3: Events participants API** only if still uncovered (`GET` per docs); otherwise skip if Events page already covers local webhooks (spec “event participants” is separate API — implement list page `/participants` only if time; else add thin section under Events)

**Spec coverage note:** implement `ListEventParticipants` client + simple page if endpoint is straightforward; else document gap in README and keep webhook events as primary “Eventos”.

- [ ] **Step 4: Commit**

```bash
git commit -m "feat: account details and audit log UI"
```

---

### Task 15: Polish, README, final verification

**Files:**

- Create/Modify: `README.md`
- Modify: flash/error mapping consistency
- Fix: remove unused scaffold Contact demo if it confuses nav (optional; can leave linked out)

- [ ] **Step 1: README**

Sections: what it is, stack, setup (`cais install`, `cais dev`), env `APP_SECRET`, demo login, how to create Kiwify API key (client_id, client_secret, account_id), webhook public URL + tunnel note, rate limit 100/min, product create limitation.

- [ ] **Step 2: Full test suite**

```bash
export PATH="$HOME/go/bin:$PATH"
cais test
# optional: cais doctor
```

Expected: all green.

- [ ] **Step 3: Manual checklist**

- [ ] Login
- [ ] Setup credentials (can use fake + httptest not needed for manual if no real keys)
- [ ] Dashboard loads / shows API errors cleanly without keys
- [ ] Navigation all pages
- [ ] Confirm modal present on refund/payout
- [ ] Receiver accepts sample curl:

```bash
curl -sS -X POST "http://localhost:8080/webhooks/kiwify/$TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"order_status":"paid","order_id":"demo"}'
```

- [ ] **Step 4: Final commit**

```bash
git add README.md
git commit -m "docs: README and final Kiwify dashboard polish"
```

---

## Self-review (plan vs spec)

| Spec requirement                             | Task(s)               |
| -------------------------------------------- | --------------------- |
| Live proxy architecture                      | 4–5, 8–14             |
| Local login + UI setup                       | 1, 6                  |
| client_id + client_secret + account_id       | 3, 6 (amended design) |
| Encrypted secrets + APP_SECRET               | 2, 3, 5               |
| Token cache OAuth                            | 4–5                   |
| All read endpoints                           | 8–14                  |
| Refund, payout, affiliate edit, webhook CRUD | 9, 11–13              |
| Confirm + audit                              | 9, 11–13, 14          |
| Webhook receiver with token path             | 13                    |
| Events feed                                  | 13                    |
| Kiwi green PT-BR                             | 7 + pages             |
| Tests client/handlers/crypto/receiver        | 2–6, 9, 13            |
| No product create                            | 10 (read-only + note) |

**Type consistency:** `KiwifySettings`, `AuditLog`, `WebhookEvent`, `kiwify.Client`, `TokenStore`, audit action strings `sales.refund`, `finance.payout`, `affiliates.edit`, `webhooks.create|update|delete`.

**Placeholder scan:** Affiliate edit HTTP verb left as “verify at implement time” because docs path may be PUT vs POST — resolve in Task 12 against live docs; no other TBDs.

---

## Execution handoff

Plan saved to `docs/superpowers/plans/2026-07-31-kiwify-dashboard.md`.

**Two execution options:**

1. **Subagent-Driven (recommended)** — fresh subagent per task, review between tasks
2. **Inline Execution** — this session with executing-plans and checkpoints

Which approach?
