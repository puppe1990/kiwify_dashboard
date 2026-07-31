package middleware

import (
	"net/http"
	"strings"

	"github.com/puppe1990/kiwify_dashboard/internal/store"
)

// paths that never require Kiwify credentials to be configured.
var setupSkipPrefixes = []string{
	"/login",
	"/signup",
	"/forgot-password",
	"/reset-password",
	"/setup",
	"/webhooks/kiwify",
	"/static",
	"/health",
}

// RequireSetup redirects to /setup when Kiwify credentials are not configured.
// Auth, setup, webhook, health, and static paths are skipped.
func RequireSetup(st store.Store) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if skipSetupCheck(r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}
			ok, err := st.Configured()
			if err != nil {
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
			if !ok {
				http.Redirect(w, r, "/setup", http.StatusSeeOther)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func skipSetupCheck(path string) bool {
	for _, p := range setupSkipPrefixes {
		if path == p || strings.HasPrefix(path, p+"/") {
			return true
		}
	}
	// Also treat /webhooks/kiwify* (no extra slash) as skip for prefix style paths.
	if strings.HasPrefix(path, "/webhooks/kiwify") {
		return true
	}
	return false
}
