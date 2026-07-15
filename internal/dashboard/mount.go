package dashboard

import (
	"apex/strategy"
	mdweb "apex/marketdata/web"
	"log/slog"
	"net/http"
)

// Mount registers the composed home page. This is the only package allowed
// to import more than one module's web subpackage - it composes fragments
// from each, it doesn't own any domain state itself.
func Mount(mux *http.ServeMux, log *slog.Logger, mdDash *mdweb.Dashboard, stratSvc *strategy.Service) {
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		strategies, err := stratSvc.List(r.Context())
		if err != nil {
			log.Error("list strategies", slog.Any("error", err))
			http.Error(w, "failed to load strategies", http.StatusInternalServerError)
			return
		}

		if err := Index(mdDash.Active(), strategies).Render(r.Context(), w); err != nil {
			log.Error("render index", slog.Any("error", err))
		}
	})
}
