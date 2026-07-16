package providers

import (
	"apex/internal/domain"
	"context"
	"sync"
	"time"

	"github.com/alpacahq/alpaca-trade-api-go/v3/marketdata"
	"github.com/alpacahq/alpaca-trade-api-go/v3/marketdata/stream"
)

type AlpacaProvider struct {
	client     *stream.StocksClient
	historical *marketdata.Client

	mu        sync.Mutex
	listeners map[string][]barHandler
}

func NewAlpacaProvider(ctx context.Context, key, secret string) (*AlpacaProvider, error) {
	cli := stream.NewStocksClient(marketdata.IEX, stream.WithCredentials(key, secret))
	err := cli.Connect(ctx)
	if err != nil {
		return nil, err
	}

	historical := marketdata.NewClient(marketdata.ClientOpts{
		APIKey:    key,
		APISecret: secret,
		Feed:      marketdata.IEX,
	})

	return &AlpacaProvider{
		client:     cli,
		historical: historical,
		listeners:  make(map[string][]barHandler),
	}, nil
}

func (ap *AlpacaProvider) Name() string { return "alpaca" }

// Subscribe registers handler for symbol. Any number of distinct symbols can
// be subscribed concurrently: ap.dispatch is a stable bound method passed to
// the SDK on every call, so repeat calls extend Alpaca's additive
// subscribed-symbol set rather than replacing the client's single registered
// handler.
func (ap *AlpacaProvider) Subscribe(ctx context.Context, symbol string, handler barHandler) error {
	ap.mu.Lock()
	ap.listeners[symbol] = append(ap.listeners[symbol], handler)
	ap.mu.Unlock()

	return ap.client.SubscribeToBars(ap.dispatch, symbol)
}

func (ap *AlpacaProvider) dispatch(b stream.Bar) {
	ap.mu.Lock()
	handlers := append([]barHandler(nil), ap.listeners[b.Symbol]...)
	ap.mu.Unlock()

	bar := domain.NewBar(b.Timestamp, b.Symbol, b.High, b.Open, b.Low, b.Close, b.Volume)
	for _, h := range handlers {
		h(bar)
	}
}

func (ap *AlpacaProvider) Unsubscribe(ctx context.Context, symbol string) error {
	ap.mu.Lock()
	delete(ap.listeners, symbol)
	ap.mu.Unlock()

	return ap.client.UnsubscribeFromBars(symbol)
}

// GetBackfillBars always fetches 1-minute bars - the only resolution ever
// stored. Coarser display intervals are computed on read via TimescaleDB
// time_bucket over this 1-minute data (see Module.ListBars).
func (ap *AlpacaProvider) GetBackfillBars(ctx context.Context, symbol string, start, end time.Time) ([]domain.Bar, error) {
	bars, err := ap.historical.GetBars(symbol, marketdata.GetBarsRequest{
		TimeFrame: marketdata.NewTimeFrame(1, marketdata.Min),
		Start:     start,
		End:       end,
	})
	if err != nil {
		return nil, err
	}

	finalBars := make([]domain.Bar, 0, len(bars))
	for _, bar := range bars {
		finalBars = append(finalBars, domain.NewBar(bar.Timestamp, symbol, bar.High, bar.Open, bar.Low, bar.Close, bar.Volume))
	}

	return finalBars, nil
}
