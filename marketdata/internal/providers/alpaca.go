package providers

import (
	"apex/internal/domain"
	"context"
	"sync"
	"time"

	"github.com/alpacahq/alpaca-trade-api-go/v3/marketdata"
	"github.com/alpacahq/alpaca-trade-api-go/v3/marketdata/stream"
)

type barListener struct {
	tf      string
	handler barHandler
}

type AlpacaProvider struct {
	client     *stream.StocksClient
	historical *marketdata.Client

	mu        sync.Mutex
	listeners map[string][]barListener
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
		listeners:  make(map[string][]barListener),
	}, nil
}

func (ap *AlpacaProvider) Name() string { return "alpaca" }

// Subscribe registers handler for symbol/tf. Multiple timeframes for the same
// symbol, and any number of distinct symbols, can be subscribed concurrently:
// ap.dispatch is a stable bound method passed to the SDK on every call, so
// repeat calls extend Alpaca's additive subscribed-symbol set rather than
// replacing the client's single registered handler.
func (ap *AlpacaProvider) Subscribe(ctx context.Context, symbol string, tf string, handler barHandler) error {
	ap.mu.Lock()
	ap.listeners[symbol] = append(ap.listeners[symbol], barListener{tf: tf, handler: handler})
	ap.mu.Unlock()

	return ap.client.SubscribeToBars(ap.dispatch, symbol)
}

func (ap *AlpacaProvider) dispatch(b stream.Bar) {
	ap.mu.Lock()
	listeners := append([]barListener(nil), ap.listeners[b.Symbol]...)
	ap.mu.Unlock()

	for _, l := range listeners {
		bar := domain.NewBar(b.Timestamp, b.Symbol, l.tf, b.High, b.Open, b.Low, b.Close, b.Volume)
		l.handler(bar)
	}
}

func (ap *AlpacaProvider) Unsubscribe(ctx context.Context, symbol string, tf string) error {
	ap.mu.Lock()
	kept := make([]barListener, 0, len(ap.listeners[symbol]))
	for _, l := range ap.listeners[symbol] {
		if l.tf != tf {
			kept = append(kept, l)
		}
	}
	if len(kept) == 0 {
		delete(ap.listeners, symbol)
	} else {
		ap.listeners[symbol] = kept
	}
	empty := len(kept) == 0
	ap.mu.Unlock()

	if empty {
		return ap.client.UnsubscribeFromBars(symbol)
	}
	return nil
}

func (ap *AlpacaProvider) GetBackfillBars(ctx context.Context, symbol string, tf string, start, end time.Time) ([]domain.Bar, error) {
	bars, err := ap.historical.GetBars(symbol, marketdata.GetBarsRequest{
		TimeFrame: ap.parseTimeFrame(tf),
		Start:     start,
		End:       end,
	})
	if err != nil {
		return nil, err
	}

	finalBars := make([]domain.Bar, 0, len(bars))
	for _, bar := range bars {
		finalBars = append(finalBars, domain.NewBar(bar.Timestamp, symbol, tf, bar.High, bar.Open, bar.Low, bar.Close, bar.Volume))
	}

	return finalBars, nil
}

func (ap *AlpacaProvider) parseTimeFrame(tf string) marketdata.TimeFrame {
	switch tf {
	case "1Min":
		return marketdata.NewTimeFrame(1, marketdata.Min)
	case "5min":
		return marketdata.NewTimeFrame(5, marketdata.Min)
	default:
		return marketdata.NewTimeFrame(1, marketdata.Day)
	}
}
