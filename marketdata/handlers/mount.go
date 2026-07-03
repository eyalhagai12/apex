package handlers

import (
	"apex/internal/httputil"
	"apex/marketdata"
	"net/http"
)

func Mount(mux *http.ServeMux, mkdata *marketdata.Module) {
	h := New(mkdata)
	mux.HandleFunc("POST /marketdata/subscribe", httputil.Wrap(h.SubscribeToSymbol))
}
