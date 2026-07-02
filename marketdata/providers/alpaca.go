package providers

import (
	"apex/internal/domain"
	"context"
	"time"

	"github.com/alpacahq/alpaca-trade-api-go/v3/marketdata"
	"github.com/alpacahq/alpaca-trade-api-go/v3/marketdata/stream"
)

type AlpacaProvider struct {
	client     *stream.StocksClient
	historical *marketdata.Client
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
	}, nil
}

func (ap *AlpacaProvider) Name() string { return "alpaca" }

func (ap *AlpacaProvider) Subscribe(ctx context.Context, symbol string, tf string, handler barHandler) error {
	return ap.client.SubscribeToBars(func(b stream.Bar) {
		bar := domain.NewBar(b.Timestamp, symbol, tf, b.High, b.Open, b.Low, b.Close, b.Volume)
		handler(bar)
	}, symbol)
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
