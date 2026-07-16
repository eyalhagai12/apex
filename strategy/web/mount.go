package web

import (
	"apex/strategy"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

const maxUploadSize = 10 << 20 // 10MB

type handlers struct {
	svc *strategy.Service
	log *slog.Logger
}

// Mount registers strategy's own htmx routes (create).
func Mount(mux *http.ServeMux, log *slog.Logger, svc *strategy.Service) {
	h := &handlers{svc: svc, log: log}
	mux.HandleFunc("POST /web/strategies", h.handleCreate)
	mux.HandleFunc("DELETE /web/strategies/{id}", h.handleDelete)
}

func (h *handlers) handleCreate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
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

	file, _, err := r.FormFile("file")
	if err != nil {
		if renderErr := CreateError("strategy file is required").Render(r.Context(), w); renderErr != nil {
			h.log.Error("render create error", slog.Any("error", renderErr))
		}
		return
	}
	defer file.Close()

	code, err := io.ReadAll(file)
	if err != nil {
		h.log.Error("read strategy file", slog.String("name", name), slog.Any("error", err))
		if renderErr := CreateError("failed to read uploaded file").Render(r.Context(), w); renderErr != nil {
			h.log.Error("render create error", slog.Any("error", renderErr))
		}
		return
	}

	s, err := h.svc.Create(r.Context(), name, code)
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

func (h *handlers) handleDelete(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		if renderErr := CreateError("invalid strategy id").Render(r.Context(), w); renderErr != nil {
			h.log.Error("render create error", slog.Any("error", renderErr))
		}
		return
	}

	h.svc.Delete(r.Context(), id)

}
