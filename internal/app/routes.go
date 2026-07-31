package app

import (
	"net/http"

	"github.com/puppe1990/cais/pkg/cais"
	"github.com/puppe1990/cais/pkg/cais/middleware"
	"github.com/puppe1990/kiwify_dashboard/internal/handlers"
)

func registerRoutes(r *cais.Router, deps Deps, cfg cais.Config) {
	home := handlers.NewHomeHandler(deps.Renderer, deps.Site, deps.Catalog, cfg, deps.Inertia)
	contact := handlers.NewContactHandler(deps.Renderer, deps.Store, deps.Site, deps.Catalog, cfg, deps.Inertia)
	dashboard := handlers.NewDashboardHandler(deps.Renderer, deps.Store, deps.AppSecret, deps.Site, cfg, deps.Inertia)
	auth := handlers.NewAuthHandler(deps.Renderer, deps.Store, deps.Site, deps.Store.Sessions(), cfg, deps.Catalog, deps.Inertia)
	setup := handlers.NewSetupHandler(deps.Store, deps.AppSecret, deps.Site, cfg, deps.Inertia)
	settings := handlers.NewSettingsHandler(deps.Store, deps.AppSecret, deps.Site, cfg, deps.Inertia)
	sales := handlers.NewSalesHandler(deps.Store, deps.AppSecret, deps.Site, cfg, deps.Inertia, nil)

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

	r.Get("/setup", middleware.RequireAuthFunc("/login", setup.Get))
	r.Post("/setup", middleware.RequireAuthFunc("/login", setup.Post))
	r.Get("/settings", middleware.RequireAuthFunc("/login", settings.Get))
	r.Post("/settings", middleware.RequireAuthFunc("/login", settings.Post))
	r.Get("/dashboard", middleware.RequireAuthFunc("/login", dashboard.ServeHTTP))
	r.Get("/sales", middleware.RequireAuthFunc("/login", sales.List))
	r.Get("/sales/{id}", middleware.RequireAuthFunc("/login", cais.StringParam("id", sales.Show)))
	r.Post("/sales/{id}/refund", middleware.RequireAuthFunc("/login", cais.StringParam("id", sales.Refund)))
}
