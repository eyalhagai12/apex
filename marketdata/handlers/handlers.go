package handlers

import (
	"apex/marketdata"
	"net/http"
)

type Handlers struct {
	mkdata *marketdata.Module
}

func New(mkdata *marketdata.Module) *Handlers {
	return &Handlers{mkdata: mkdata}
}

type SubscribeRequest struct {
	TimeFrame string `json:"timeframe"`
	Symbol    string `json:"symbol"`
}

type SubscribeResponse struct {
	TimeFrame string `json:"timeframe"`
	Symbol    string `json:"symbol"`
}

func (h *Handlers) SubscribeToSymbol(w http.ResponseWriter, r *http.Request, req SubscribeRequest) (SubscribeResponse, int, error) {
	if err := h.mkdata.Subscribe(r.Context(), req.Symbol, req.TimeFrame); err != nil {
		return SubscribeResponse{}, http.StatusInternalServerError, err
	}

	return SubscribeResponse{TimeFrame: req.TimeFrame, Symbol: req.Symbol}, http.StatusOK, nil
}
