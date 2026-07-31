package app

import (
	"net/http"

	"github.com/puppe1990/cais/pkg/cais"
	"github.com/puppe1990/cais/pkg/cais/middleware"

	"github.com/puppe1990/kiwify_dashboard/internal/handlers"
)

func registerRoutes(r *cais.Router, deps Deps, cfg cais.Config) {
	home := handlers.NewHomeHandlerWithStore(deps.Renderer, deps.Store, deps.Site, deps.Catalog, cfg, deps.Inertia)
	contact := handlers.NewContactHandler(deps.Renderer, deps.Store, deps.Site, deps.Catalog, cfg, deps.Inertia)
	dashboard := handlers.NewDashboardHandler(deps.Renderer, deps.Store, deps.AppSecret, deps.Site, cfg, deps.Inertia)
	auth := handlers.NewAuthHandler(deps.Renderer, deps.Store, deps.Site, deps.Store.Sessions(), cfg, deps.Catalog, deps.Inertia)
	setup := handlers.NewSetupHandler(deps.Store, deps.AppSecret, deps.Site, cfg, deps.Inertia)
	settings := handlers.NewSettingsHandler(deps.Store, deps.AppSecret, deps.Site, cfg, deps.Inertia)
	sales := handlers.NewSalesHandler(deps.Store, deps.AppSecret, deps.Site, cfg, deps.Inertia, nil)
	products := handlers.NewProductsHandler(deps.Store, deps.AppSecret, deps.Site, cfg, deps.Inertia, nil)
	finance := handlers.NewFinanceHandler(deps.Store, deps.AppSecret, deps.Site, cfg, deps.Inertia, nil)
	affiliates := handlers.NewAffiliatesHandler(deps.Store, deps.AppSecret, deps.Site, cfg, deps.Inertia, nil)
	webhooks := handlers.NewWebhooksHandler(deps.Store, deps.AppSecret, deps.Site, cfg, deps.Inertia, nil)
	events := handlers.NewEventsHandler(deps.Store, deps.Site, cfg, deps.Inertia)
	account := handlers.NewAccountHandler(deps.Store, deps.AppSecret, deps.Site, cfg, deps.Inertia, nil)
	audit := handlers.NewAuditHandler(deps.Store, deps.Site, cfg, deps.Inertia)
	kiwifyWebhook := handlers.NewKiwifyWebhookHandler(deps.Store)

	loginLimit := middleware.NewRateLimiter(10, cfg)
	resetLimit := middleware.NewRateLimiter(10, cfg)
	contactLimit := middleware.NewRateLimiter(20, cfg)

	r.Get("/", home.ServeHTTP)
	r.Get("/contact", contact.Get)
	r.Post("/contact", contactLimit.Middleware(http.HandlerFunc(contact.Post)).ServeHTTP)
	r.Get("/login", auth.Login)
	r.Post("/login", loginLimit.Middleware(http.HandlerFunc(auth.LoginPost)).ServeHTTP)
	r.Get("/signup", auth.SignUp)
	r.Post("/signup", loginLimit.Middleware(http.HandlerFunc(auth.SignUpPost)).ServeHTTP)
	r.Get("/forgot-password", auth.ForgotPassword)
	r.Post("/forgot-password", resetLimit.Middleware(http.HandlerFunc(auth.ForgotPasswordPost)).ServeHTTP)
	r.Get("/reset-password", auth.ResetPassword)
	r.Post("/reset-password", resetLimit.Middleware(http.HandlerFunc(auth.ResetPasswordPost)).ServeHTTP)
	r.Post("/logout", auth.LogoutPost)

	// Public Kiwify receiver — no RequireAuth / must stay outside auth group.
	// RequireSetup already skips /webhooks/kiwify prefix.
	r.Post("/webhooks/kiwify/{token}", cais.StringParam("token", kiwifyWebhook.Receive))

	r.Get("/setup", middleware.RequireAuthFunc("/login", setup.Get))
	r.Post("/setup", middleware.RequireAuthFunc("/login", setup.Post))
	r.Get("/settings", middleware.RequireAuthFunc("/login", settings.Get))
	r.Post("/settings", middleware.RequireAuthFunc("/login", settings.Post))
	r.Get("/dashboard", middleware.RequireAuthFunc("/login", dashboard.ServeHTTP))
	r.Get("/sales", middleware.RequireAuthFunc("/login", sales.List))
	r.Get("/sales/{id}", middleware.RequireAuthFunc("/login", cais.StringParam("id", sales.Show)))
	r.Post("/sales/{id}/refund", middleware.RequireAuthFunc("/login", cais.StringParam("id", sales.Refund)))
	r.Get("/products", middleware.RequireAuthFunc("/login", products.List))
	r.Get("/products/{id}", middleware.RequireAuthFunc("/login", cais.StringParam("id", products.Show)))
	r.Get("/finance", middleware.RequireAuthFunc("/login", finance.Get))
	r.Post("/finance/payouts", middleware.RequireAuthFunc("/login", finance.CreatePayout))
	r.Get("/affiliates", middleware.RequireAuthFunc("/login", affiliates.List))
	r.Get("/affiliates/{id}", middleware.RequireAuthFunc("/login", cais.StringParam("id", affiliates.Show)))
	r.Post("/affiliates/{id}", middleware.RequireAuthFunc("/login", cais.StringParam("id", affiliates.Update)))

	// Webhooks CRUD + local events feed (auth required).
	// Register static /webhooks/new before /webhooks/{id}.
	r.Get("/webhooks", middleware.RequireAuthFunc("/login", webhooks.List))
	r.Get("/webhooks/new", middleware.RequireAuthFunc("/login", webhooks.New))
	r.Post("/webhooks", middleware.RequireAuthFunc("/login", webhooks.Create))
	r.Get("/webhooks/{id}", middleware.RequireAuthFunc("/login", cais.StringParam("id", webhooks.Show)))
	r.Post("/webhooks/{id}", middleware.RequireAuthFunc("/login", cais.StringParam("id", webhooks.Update)))
	// DELETE (not POST …/delete): avoids conflict with POST /webhooks/kiwify/{token}
	// under Go ServeMux overlapping wildcards.
	r.Delete("/webhooks/{id}", middleware.RequireAuthFunc("/login", cais.StringParam("id", webhooks.Delete)))
	r.Get("/events", middleware.RequireAuthFunc("/login", events.List))
	r.Get("/account", middleware.RequireAuthFunc("/login", account.Get))
	r.Get("/audit", middleware.RequireAuthFunc("/login", audit.List))
}
