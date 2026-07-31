# kiwify_dashboard

Full-stack Go app built with [Cais](https://github.com/puppe1990/cais): Inertia.js, Svelte, Tailwind, and SQLite.

## Stack

- Go 1.26 (net/http stdlib) + gonertia
- Svelte 5 + @inertiajs/svelte (Vite → web/static/build/)
- Tailwind CSS 3.x
- SQLite (modernc.org/sqlite, no CGO)

## Quick start

```bash
cais install  # npm install + go mod tidy
cais dev        # http://localhost:8080
cais test       # full test suite
cais build      # bin/server
```

## Cais CLI

This app was scaffolded with the Cais CLI. Useful commands:

```bash
cais install               # npm install + go mod tidy
cais css                   # build Tailwind
cais dev                   # air + tailwind + vite build --watch (Svelte)
cais server                # go run ./cmd/server
cais console               # interactive Go REPL + SQL
cais g handler <name>      # handler + test + page template
cais g resource <name>     # model + migration + admin CRUD
cais g page <name>         # page template only
cais g migration <name>    # SQL migration file
cais test                  # go test ./...
cais doctor                # verify setup
```

## CI and pre-commit

GitHub Actions runs Go tests, `golangci-lint`, Prettier, and `npm test` on every push/PR to `main`.

```bash
make pre-commit-install   # once: installs git hooks
make ci                   # test + lint + format-check locally
```

Pre-commit hooks run: trailing whitespace, Prettier, `go fmt`, `go test`, `golangci-lint`, and `npm test`.

## Structure

```
pkg/cais/          → framework (via dependency)
internal/app/      → bootstrap and routes
internal/handlers/ → HTTP handlers
internal/store/    → SQLite + migrations
web/templates/app.html → Inertia root shell
web/src/pages/       → Svelte pages
web/static/          → CSS + Vite build + PWA
cmd/server/        → entry point
```

## Environment variables

| Variable  | Default         | Description      |
| --------- | --------------- | ---------------- |
| PORT      | :8080           | Server port      |
| DB_PATH   | ./data/app.db   | SQLite file path |
| ENV       | development     | Environment      |

Health check: GET /health → {"status":"ok"}

## Testing on phone (LAN)

1. Run `cais dev` and note the **LAN** URL printed at boot (e.g. `http://192.168.1.10:8080`).
2. Open that URL in mobile Safari/Chrome on the same Wi‑Fi.
3. After template or SSE changes, run `cais pwa --bump` and reinstall the PWA (or clear site data) so the service worker cache refreshes.
4. Run `cais doctor --mobile` to catch flash markup, font CSP, and SW cache issues.
