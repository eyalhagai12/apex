package web

import (
	"apex/strategy"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
)

type handlers struct {
	svc *strategy.Service
	log *slog.Logger
}

// Mount registers strategy's own htmx routes (create).
func Mount(mux *http.ServeMux, log *slog.Logger, svc *strategy.Service) {
	h := &handlers{svc: svc, log: log}
	mux.HandleFunc("POST /web/strategies", h.handleCreate)
}

func (h *handlers) handleCreate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		if renderErr := CreateError("invalid form data").Render(r.Context(), w); renderErr != nil {
			h.log.Error("render create error", slog.Any("error", renderErr))
		}
		return
	}

	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		if err := CreateError("name is required").Render(r.Context(), w); err != nil {
			h.log.Error("render create error", slog.Any("error", err))
		}
		return
	}

	s, err := h.svc.Create(r.Context(), name)
	if err != nil {
		h.log.Error("create strategy", slog.String("name", name), slog.Any("error", err))
		if renderErr := CreateError(fmt.Sprintf("failed to create %s: %v", name, err)).Render(r.Context(), w); renderErr != nil {
			h.log.Error("render create error", slog.Any("error", renderErr))
		}
		return
	}

	if err := Row(s).Render(r.Context(), w); err != nil {
		h.log.Error("render strategy row", slog.Any("error", err))
	}
}
