package handlers

import (
	"context"
	"net/http"

	"github.com/puppe1990/cais/pkg/cais"
	"github.com/puppe1990/cais/pkg/cais/flash"
	inertia "github.com/romsar/gonertia/v3"

	"github.com/puppe1990/kiwify_dashboard/internal/kiwify"
	"github.com/puppe1990/kiwify_dashboard/internal/store"
)

// flashProps returns an Inertia flash map from the cais flash cookie/context, if any.
func flashProps(r *http.Request) inertia.Flash {
	if msg, ok := flash.MessageFromRequest(r); ok {
		return inertia.Flash{msg.Kind: msg.Message}
	}
	return nil
}

// redirectWithFlash stores a one-shot cookie flash (cais middleware) and redirects.
// inertia.SetFlash alone does nothing here: Inertia is created without a flash provider.
func redirectWithFlash(w http.ResponseWriter, r *http.Request, i *inertia.Inertia, cfg cais.Config, kind, message, location string) {
	flash.Set(w, kind, message, cfg.CookieSecure())
	ctx := inertia.SetFlash(r.Context(), inertia.Flash{kind: message})
	i.Redirect(w, r.WithContext(ctx), location, http.StatusSeeOther)
}

// probeKiwifyOAuth obtains an OAuth token with the stored credentials.
// Overridden in tests to avoid real network calls.
var probeKiwifyOAuth = func(ctx context.Context, st store.Store, key []byte) error {
	client, err := kiwify.NewClientFromStore(st, key, nil)
	if err != nil {
		return err
	}
	_, err = client.GetToken(ctx)
	return err
}
