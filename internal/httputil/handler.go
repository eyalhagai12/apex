package httputil

import (
	"encoding/json"
	"net/http"
)

type HandlerFunc[Req, Res any] func(w http.ResponseWriter, r *http.Request, rdata Req) (Res, int, error)

func Wrap[Req, Res any](handler HandlerFunc[Req, Res]) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req Req
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			WriteError(w, err, http.StatusBadRequest)
			return
		}

		// TODO: validation

		res, status, err := handler(w, r, req)
		if err != nil {
			WriteError(w, err, status)
			return
		}

		if err := WriteJSON(w, res, status); err != nil {
			WriteError(w, err, http.StatusInternalServerError)
			return
		}
	}
}
