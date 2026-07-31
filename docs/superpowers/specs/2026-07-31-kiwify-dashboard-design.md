# Kiwify Ops Dashboard — Design Spec

**Date:** 2026-07-31  
**Status:** Approved for implementation planning  
**Stack:** [cais](https://github.com/puppe1990/cais) (Go + Inertia.js + Svelte 5 + Tailwind + SQLite)  
**API:** [Kiwify Public API](https://docs.kiwify.com.br/api-reference/general) (`https://public-api.kiwify.com`)

## 1. Goal

Personal operational dashboard for a Kiwify producer that:

- **Reads** everything the Public API exposes (account, sales, stats, products, balances, payouts, affiliates, webhooks, event participants).
- **Acts** where the API allows: refund sale, request payout, edit affiliate, create/update/delete webhooks.
- **Receives** Kiwify webhook events into the app for a local event feed.

This is **not** multi-account/agency and **not** a product CMS (the API has no create/update product endpoints).

## 2. Users and auth

| Layer | Behavior |
| --- | --- |
| App login | Local cais session (email/password). Required for all UI and mutations. |
| Kiwify credentials | Configured in UI after first login (`/setup` / `/settings`): `client_id` + `client_secret` + `account_id` (OAuth `/oauth/token` requires client_id and client_secret). |
| API calls | Server-side only. OAuth bearer token cached and reused until near expiry (~96h). |

Flow: `login` → if settings missing → `/setup` → dashboard.

## 3. Architecture (live proxy)

```
Browser (Svelte 5 + Inertia, Kiwi green UI)
  → Go handlers (session + CSRF)
      → internal/kiwify client → https://public-api.kiwify.com
      → SQLite (settings, audit_logs, webhook_events)
Kiwify → POST /webhooks/kiwify (public) → webhook_events
```

**Approach:** every list/detail page fetches the Public API on request. No catalog sync jobs in MVP.

**Rationale:** full API coverage with minimal moving parts. Rate limit is 100 req/min — acceptable for single-user ops; hybrid cache can come later if needed.

### Out of scope (MVP)

- Create/edit/delete products (not in Public API)
- Multi-account / agency
- TTL cache / background sync of sales/products
- Typing challenge before refund (simple confirm modal + audit is enough)

## 4. Visual design

- **Direction:** Kiwi green — light mint backgrounds, green sidebar (`#14532d` / `#166534`), accent `#22c55e`, white cards with soft green borders.
- **Chrome:** left sidebar + main content; Portuguese (PT-BR) copy.
- **Density:** SaaS operational (readable tables, clear CTAs), not terminal-dark.

## 5. Screens and routes

| Route | Purpose |
| --- | --- |
| `GET /login`, auth routes | cais local auth |
| `GET/POST /setup` | First-time API key + account_id |
| `GET /` | Dashboard: sales stats, available balance, recent sales, recent local webhook events |
| `GET /sales` | List sales (date range max 90 days per API) |
| `GET /sales/{id}` | Sale detail |
| `POST /sales/{id}/refund` | Refund (confirm + audit) |
| `GET /products`, `GET /products/{id}` | Read-only product list/detail |
| `GET /finance` | Balances + payouts list |
| `POST /finance/payouts` | Request payout (confirm + audit) |
| `GET /affiliates`, `GET /affiliates/{id}` | List/detail |
| `POST /affiliates/{id}` | Edit affiliate (confirm + audit) |
| `GET /webhooks`, CRUD routes | Manage webhooks via API |
| `GET /events` | Local feed from `webhook_events` |
| `GET /audit` | Sensitive action history |
| `GET /account` | Account details from API |
| `GET/POST /settings` | View/update Kiwify credentials |
| `POST /webhooks/kiwify/{receive_token}` | Public receiver (no session; token gate) |

All app routes except login and webhook receiver use `RequireAuth`.

## 6. Local data model

Beyond cais defaults (`users`, `sessions`):

### `kiwify_settings` (singleton row)

- `account_id` (text)
- `client_id` (text)
- `client_secret_ciphertext` (blob/text)
- `oauth_access_token_ciphertext` (nullable)
- `token_expires_at` (nullable datetime)
- `webhook_receive_token` (text, random, used in receiver URL)
- `updated_at`

Encryption: symmetric key derived from env `APP_SECRET` (required in production; generated for local dev if missing only when documented in README). Plain secrets never appear in Inertia props (settings UI shows masked secret + “replace” field).

### `audit_logs`

- `user_id`, `action`, `resource_type`, `resource_id`
- `request_summary` (JSON-ish text without full secrets)
- `response_status`, `response_body` (truncated)
- `ip`, `created_at`

Written for: refund, payout create, affiliate edit, webhook create/update/delete — success **and** failure.

### `webhook_events`

- `event_type`, `payload_json`, `headers_json`
- `received_at`, `processed_ok` (bool)

Retention: keep last N or all in MVP (simple list + pagination); pruning can wait.

## 7. Kiwify client (`internal/kiwify`)

Responsibilities:

1. **Token management:** `GetToken(ctx)` returns valid bearer; if missing/expired, `POST /v1/oauth/token` with form `client_id` + `client_secret`, persist new token + expiry (`expires_in` seconds).
2. **Request helper:** base URL `https://public-api.kiwify.com/v1`, headers:
   - `Authorization: Bearer <token>`
   - `x-kiwify-account-id: <account_id>`
3. **Domain methods** aligned with docs:
   - Account details
   - Sales list/single/stats/refund
   - Products list/single
   - Finance balances (list/single), payouts (list/single/create)
   - Affiliates list/single/edit
   - Webhooks list/single/create/edit/delete
   - Events participants list
4. **Errors:** typed mapping for 400, 401/403, 429, 5xx → user-facing PT-BR messages; handlers flash or page error props.

Rate limit: surface clear error; do not spin retry loops that amplify 429.

## 8. Sensitive actions UX

1. User opens confirm modal with human summary (order id, amount, buyer email / payout amount / webhook URL).
2. Confirm → `POST` with CSRF.
3. Handler calls client → writes `audit_logs` → Inertia flash + `303` redirect.
4. Failures still audited; flash explains API error.

## 9. Webhook receiver

- Public `POST /webhooks/kiwify/{receive_token}` (token generated at setup and stored in `kiwify_settings`; rejects wrong/missing token with 404).
- Parse JSON body; store raw payload + selected headers + inferred `event_type`.
- Respond `200` quickly; do not call Public API inside the receiver path.

Managing webhooks (CRUD) remains via Public API. Settings UI shows the full public receive URL so the producer can register it in Kiwify (deploy or tunnel).

## 10. Package layout

```
cmd/server/
internal/
  app/           # routes, DI
  handlers/      # one area per domain + kiwify_webhook
  kiwify/        # HTTP client, token, types, errors
  store/         # settings, audit, webhook_events
  crypto/        # encrypt/decrypt
web/src/
  pages/         # Inertia pages
  layouts/       # AppLayout (sidebar)
  components/    # StatCard, DataTable, ConfirmModal, FlashBanner
migrations/      # settings, audit_logs, webhook_events
```

Scaffold with `cais new` (or fill current empty project directory), then generators + hand-written kiwify client and UI.

## 11. Testing strategy

| Area | What |
| --- | --- |
| `internal/kiwify` | httptest: token reuse/refresh, required headers, error decoding, representative methods |
| Action handlers | Auth required; audit row on success/failure; no secret leakage |
| Webhook receiver | Valid JSON stored; bad body → 4xx; 200 on success |
| Crypto/settings | Round-trip encrypt; Inertia props never include raw `client_secret` |

Prefer table-driven Go tests; follow cais patterns for handler tests.

## 12. Definition of done (MVP)

- [ ] App boots with `cais dev` (or project equivalent)
- [ ] Login + setup credentials + encrypted persistence
- [ ] Dashboard with stats, balance, recent sales, recent events
- [ ] Full read coverage of Public API resources above
- [ ] Mutations: refund, payout, affiliate edit, webhook CRUD — confirm + audit
- [ ] Webhook receiver (`/webhooks/kiwify/{token}`) + Events page + URL shown in settings
- [ ] Kiwi green UI, PT-BR, readable API errors
- [ ] Priority unit tests green

## 13. Implementation notes

- API dates: ISO 8601 as documented; sales list window max 90 days.
- Never log plaintext `client_secret` or full bearer tokens.
- Prefer small handlers that only orchestrate; business HTTP lives in `internal/kiwify`.
- UI: Svelte 5 + `useForm` as reactive object (cais convention); no `$form` store pattern.

## 14. Open points resolved in brainstorm

| Topic | Decision |
| --- | --- |
| Scope | Personal ops (A) + actions (C), all API modules |
| Auth | UI setup + local login |
| Sensitive actions | Confirm modal + audit log |
| Webhooks | CRUD + receive into SQLite |
| Visual | Kiwi green |
| Data strategy | Live proxy (#1) |
