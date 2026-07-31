package app

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/puppe1990/cais/pkg/cais"
	"github.com/puppe1990/cais/pkg/cais/devlog"
	"github.com/puppe1990/cais/pkg/cais/i18n"
	"github.com/puppe1990/cais/pkg/cais/meta"
	"github.com/puppe1990/cais/pkg/cais/middleware"
	"github.com/puppe1990/cais/pkg/cais/netutil"
	"github.com/puppe1990/kiwify_dashboard/internal/crypto"
	appmw "github.com/puppe1990/kiwify_dashboard/internal/middleware"
	"github.com/puppe1990/kiwify_dashboard/internal/store"
	inertia "github.com/romsar/gonertia/v3"
)

// DevAppSecretDefault is used only when ENV=development and APP_SECRET is unset.
// Never use this value for real credentials or production data.
const DevAppSecretDefault = "kiwify-ops-dev-app-secret-not-for-production"

type Deps struct {
	Renderer *cais.Renderer
	Store    store.Store
	// AppSecret is the 32-byte AES key derived via crypto.DeriveKey(APP_SECRET).
	// Use it for encrypting Kiwify client secrets and OAuth tokens — never the raw env string.
	AppSecret []byte
	StaticDir string
	Site      meta.Site
	Catalog   *i18n.Catalog
	Inertia   *inertia.Inertia
}

// ResolveAppSecret derives the encryption key from the APP_SECRET env value.
// If secret is empty and env is "development", logs a warning and uses DevAppSecretDefault.
// In any other environment an empty secret is a fatal configuration error.
func ResolveAppSecret(env, secret string) ([]byte, error) {
	if secret == "" {
		if env == "development" {
			log.Println("warning: APP_SECRET is empty; using insecure development default. Set APP_SECRET for real data.")
			secret = DevAppSecretDefault
		} else {
			return nil, fmt.Errorf("APP_SECRET is required when ENV=%s (set a long random secret)", env)
		}
	}
	return crypto.DeriveKey(secret), nil
}

type App struct {
	config cais.Config
	store  store.Store
	router *cais.Router
	server *http.Server
}

const defaultInertiaRoot = `<!DOCTYPE html>
<html lang="en">
<head><meta charset="UTF-8" />{{ .inertiaHead }}</head>
<body>{{ .inertia }}</body>
</html>`

func New(cfg cais.Config, deps Deps) (*App, error) {
	if deps.Renderer == nil {
		return nil, fmt.Errorf("renderer is required")
	}
	if deps.Store == nil {
		return nil, fmt.Errorf("store is required")
	}

	if deps.Inertia == nil {
		var err error
		deps.Inertia, err = inertia.New(defaultInertiaRoot)
		if err != nil {
			return nil, fmt.Errorf("inertia: %w", err)
		}
	}

	site := deps.Site
	if site.AppName == "" {
		site = meta.SiteFrom("kiwify_dashboard", cfg.AppURL)
	}
	site.Env = cfg.Env
	deps.Site = site

	r := cais.NewRouter()
	r.Use(middleware.CSRF(cfg))
	r.Use(middleware.LoadSession(deps.Store.Sessions()))
	r.Use(middleware.Flash)
	buf := devlog.Prepare(cfg.Env)
	if buf != nil {
		r.Use(middleware.LoggerTo(cfg, devlog.MirrorDefault(log.Writer())))
	} else {
		r.Use(middleware.Logger(cfg))
	}
	r.Use(middleware.Recover)
	r.Use(middleware.SecurityHeaders(cfg))
	// Redirect to /setup until Kiwify credentials are configured (skips auth/webhook/static).
	r.Use(appmw.RequireSetup(deps.Store))
	r.StaticForEnv("/static", deps.StaticDir, cfg)

	registerRoutes(r, deps, cfg)
	devlog.Register(r, cfg.Env, buf)
	r.Get("/health", healthHandler(deps.Store, cfg))

	return &App{
		config: cfg,
		store:  deps.Store,
		router: r,
		server: &http.Server{
			Addr:              cfg.Port,
			Handler:           r,
			ReadHeaderTimeout: 10 * time.Second,
			ReadTimeout:       30 * time.Second,
			WriteTimeout:      30 * time.Second,
			IdleTimeout:       60 * time.Second,
		},
	}, nil
}

func healthHandler(s store.Store, cfg cais.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		status := "ok"
		code := http.StatusOK
		if err := s.Ping(); err != nil {
			status = "degraded"
			code = http.StatusServiceUnavailable
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(code)
		_ = json.NewEncoder(w).Encode(netutil.HealthPayload(status, cfg.Port))
	}
}

func (a *App) Handler() http.Handler {
	return a.router
}

func (a *App) Run() error {
	return a.RunContext(context.Background())
}

func (a *App) RunContext(ctx context.Context) error {
	errCh := make(chan error, 1)
	go func() {
		errCh <- a.server.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := a.server.Shutdown(shutdownCtx); err != nil {
			_ = a.store.Close()
			return err
		}
		<-errCh
		_ = a.store.Close()
		return nil
	case err := <-errCh:
		_ = a.store.Close()
		if err == http.ErrServerClosed {
			return nil
		}
		return err
	}
}
