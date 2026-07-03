package handlers

import (
	"apex/internal/httputil"
	"apex/marketdata"
	"log/slog"
	"net/http"
)

func Mount(mux *http.ServeMux, log *slog.Logger, mkdata *marketdata.Module) {
	h := New(log, mkdata)
	mux.HandleFunc("POST /marketdata/subscribe", httputil.Wrap(h.SubscribeToSymbol))
	mux.HandleFunc("POST /marketdata/unsubscribe", httputil.Wrap(h.Unsubscribe))
	mux.HandleFunc("POST /marketdata/backfill", httputil.Wrap(h.Backfill))
}
