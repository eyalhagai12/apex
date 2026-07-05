package web

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

func Mount(mux *http.ServeMux, log *slog.Logger) {
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		if err := Index().Render(r.Context(), w); err != nil {
			log.Error("render index", slog.Any("error", err))
		}
	})

	mux.HandleFunc("GET /web/events", func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming unsupported", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-r.Context().Done():
				return
			case t := <-ticker.C:
				fmt.Fprintf(w, "data: server time is %s\n\n", t.Format(time.RFC3339))
				flusher.Flush()
			}
		}
	})
}
