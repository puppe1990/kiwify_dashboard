package main

import (
	"log"

	"github.com/puppe1990/cais/pkg/cais"
	"github.com/puppe1990/cais/pkg/cais/console"

	"github.com/puppe1990/kiwify_dashboard/internal/store"
)

func openStore(cfg cais.Config) (*store.SQLiteStore, error) {
	return store.NewSQLiteStore(cfg.DBPath, cfg.Env)
}

func bindings(s *store.SQLiteStore) map[string]any {
	return map[string]any{
		"store": s,
		"db":    s.DB(),
	}
}

func main() {
	cfg := cais.Load()
	s, err := openStore(cfg)
	if err != nil {
		log.Fatal(err)
	}

	active := s
	runErr := console.Run(console.Options{
		AppName:  "kiwify_dashboard",
		Config:   cfg,
		Bindings: bindings(active),
		Reload: func() (map[string]any, error) {
			_ = active.Close()
			next, err := openStore(cfg)
			if err != nil {
				return nil, err
			}
			active = next
			return bindings(active), nil
		},
	})
	// Close after Run so we do not log.Fatal under an active defer (gocritic exitAfterDefer).
	_ = active.Close()
	if runErr != nil {
		log.Fatal(runErr)
	}
}
