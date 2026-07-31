# Kiwify Ops Dashboard

Personal operations dashboard for [Kiwify](https://kiwify.com.br): a **live proxy** of the [Public API](https://docs.kiwify.com.br/api-reference/general) with local login, encrypted credentials, audit logs, and a secure webhook receiver.

This is a single-operator tool (not multi-account / agency). Product create/update is **not** available via the Public API — the products pages are read-only.

## Stack

- **Go** (stdlib `net/http`) via [cais](https://github.com/puppe1990/cais)
- **Inertia.js** + **Svelte 5** (Vite → `web/static/build/`)
- **Tailwind CSS** 3.x (Kiwi green UI, PT-BR)
- **SQLite** (`modernc.org/sqlite`, no CGO) — settings, audit logs, webhook events only

API base: `https://public-api.kiwify.com/v1`

## Setup

```bash
export PATH="$HOME/go/bin:$PATH"

cais install      # npm install + go mod tidy
cais db migrate   # apply SQL migrations
cais dev          # http://localhost:8080 (air + tailwind + vite watch)
```

Other useful commands:

```bash
cais test         # go test ./...
cais build        # bin/server
cais server       # go run ./cmd/server
cais doctor       # verify toolchain
```

## Environment

Copy `.env.example` → `.env` and adjust:

| Variable | Notes |
| --- | --- |
| `APP_SECRET` | **Required in production.** Symmetric material for encrypting Kiwify client secrets and OAuth tokens at rest. In development, an insecure default is used if unset (logged as a warning). |
| `ENV` | `development` / `production` |
| `PORT` | Default `:8080` |
| `APP_URL` | Public base URL (used for webhook receive URL display) |
| `DB_PATH` | Default `./data/app.db` |

Generate a strong secret for real use, e.g.:

```bash
openssl rand -hex 32
```

## Demo login

In development, the store seeds a demo user:

- **Email:** `demo@example.com`
- **Password:** `password`

After first login without Kiwify credentials you are redirected to **`/setup`**.

## Kiwify API credentials

Create an API app in the Kiwify dashboard: **Apps → API** (or equivalent Apps / API section).

You need:

1. **Client ID**
2. **Client Secret**
3. **Account ID** (`x-kiwify-account-id` header on authenticated calls)

Enter them at `/setup` (first time) or later under **Configurações** (`/settings`). The client secret is encrypted with `APP_SECRET` before storage. OAuth tokens from `POST /oauth/token` are cached encrypted until near expiry.

## Webhooks

1. Complete setup so a **receive token** is generated.
2. Copy the public URL from Settings, e.g.  
   `https://your-host/webhooks/kiwify/<token>`
3. Register that URL as a webhook in Kiwify (or via **Webhooks** in this app).

**Local development:** expose the app with a tunnel (`ngrok`, `cloudflared`, etc.) so Kiwify can reach your machine. Point `APP_URL` at the tunnel origin so the displayed receive URL is correct.

Inbound payloads are stored in local `webhook_events` and shown under **Eventos**. The receiver is public (token in path only) and skips session/setup checks.

Smoke test:

```bash
curl -sS -X POST "http://localhost:8080/webhooks/kiwify/$TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"order_status":"paid","order_id":"demo"}'
```

## Rate limit

Kiwify Public API: **100 requests/minute**. This dashboard does not add a secondary cache — every page load hits the API. Avoid aggressive refresh; on HTTP 429 the UI surfaces a PT-BR message to wait and retry.

## Features (MVP)

| Area | Behavior |
| --- | --- |
| Dashboard | Stats (30d), balance, recent sales, recent webhook events |
| Vendas | List (max 90-day window), detail, refund (confirm + audit) |
| Produtos | Read-only list/detail — **no create/edit via API** |
| Financeiro | Balances, payouts list, request payout (confirm + audit) |
| Afiliados | List/detail, edit (confirm + audit) |
| Webhooks | CRUD against Public API |
| Eventos | Local feed of received webhooks |
| Auditoria | Sensitive action history (success and failure) |
| Conta | Live `GET /account` details |
| Configurações | Update credentials + webhook receive URL |

## Project layout

```
cmd/server/          HTTP entrypoint
internal/
  app/               Router, APP_SECRET resolution
  crypto/            AES-GCM helpers
  handlers/          Inertia handlers
  kiwify/            Public API client + OAuth cache
  middleware/        RequireSetup
  store/             SQLite: settings, audit, webhook_events
web/src/pages/       Svelte 5 pages
docs/superpowers/    Design spec + implementation plan
```

## Worktree / monorepo note

This checkout may live under `.worktrees/kiwify-ops` of a parent repo. Run `cais` / `go test` from this worktree root (where `go.mod` lives). Keep secrets out of git (`.env` is gitignored).

## License / scope

Personal ops tool. Not affiliated with Kiwify beyond use of the public API. No multi-tenant features planned for MVP.
