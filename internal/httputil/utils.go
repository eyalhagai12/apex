package httputil

import (
	"encoding/json"
	"net/http"
)

func WriteJSON(w http.ResponseWriter, data any, status int) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	res, err := json.Marshal(data)
	if err != nil {
		return err
	}

	_, err = w.Write(res)
	if err != nil {
		return err
	}

	return nil
}

func WriteError(w http.ResponseWriter, err error, status int) {
	WriteJSON(w, map[string]string{"message": err.Error()}, status)
}
