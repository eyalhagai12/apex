package handlers

import (
	"apex/marketdata"
	"log/slog"
	"net/http"
	"time"
)

type Handlers struct {
	mkdata *marketdata.Module
	log    *slog.Logger
}

func New(log *slog.Logger, mkdata *marketdata.Module) *Handlers {
	return &Handlers{mkdata: mkdata, log: log}
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
	h.log.Info("subscribe request", "symbol", req.Symbol, "timeframe", req.TimeFrame)
	if err := h.mkdata.Subscribe(r.Context(), req.Symbol, req.TimeFrame); err != nil {
		return SubscribeResponse{}, http.StatusInternalServerError, err
	}

	h.log.Info("subscribe completed", "symbol", req.Symbol, "timeframe", req.TimeFrame)

	return SubscribeResponse{TimeFrame: req.TimeFrame, Symbol: req.Symbol}, http.StatusOK, nil
}

type BackfillRequest struct {
	Symbol    string    `json:"symbol"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	TimeFrame string    `json:"timeframe"`
}

type BackfillResponse struct {
	Symbol    string    `json:"symbol"`
	TimeFrame string    `json:"timeframe"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
}

func (h *Handlers) Backfill(w http.ResponseWriter, r *http.Request, req BackfillRequest) (BackfillResponse, int, error) {
	h.log.Info("backfill request", "symbol", req.Symbol, "timeframe", req.TimeFrame, "start_time", req.StartTime, "end_time", req.EndTime)
	if err := h.mkdata.Backfill(r.Context(), req.Symbol, req.TimeFrame, req.EndTime, req.EndTime); err != nil {
		return BackfillResponse{}, http.StatusInternalServerError, err
	}

	h.log.Info("backfill completed", "symbol", req.Symbol, "timeframe", req.TimeFrame, "start_time", req.StartTime, "end_time", req.EndTime)

	return BackfillResponse{
		Symbol:    req.Symbol,
		TimeFrame: req.TimeFrame,
		StartTime: req.StartTime,
		EndTime:   req.EndTime,
	}, http.StatusOK, nil
}

type UnsubscribeRequest struct {
	TimeFrame string `json:"timeframe"`
	Symbol    string `json:"symbol"`
}

type UnsubscribeResponse struct {
	TimeFrame string `json:"timeframe"`
	Symbol    string `json:"symbol"`
}

func (h *Handlers) Unsubscribe(w http.ResponseWriter, r *http.Request, req UnsubscribeRequest) (UnsubscribeResponse, int, error) {
	h.log.Info("unsubscribe request", "symbol", req.Symbol, "timeframe", req.TimeFrame)
	if err := h.mkdata.Unsubscribe(r.Context(), req.Symbol, req.TimeFrame); err != nil {
		return UnsubscribeResponse{}, http.StatusInternalServerError, err
	}

	h.log.Info("unsubscribe completed", "symbol", req.Symbol, "timeframe", req.TimeFrame)

	return UnsubscribeResponse{TimeFrame: req.TimeFrame, Symbol: req.Symbol}, http.StatusOK, nil
}
