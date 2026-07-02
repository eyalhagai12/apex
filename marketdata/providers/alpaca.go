package providers

import (
	"apex/internal/domain"
	"context"

	"github.com/alpacahq/alpaca-trade-api-go/v3/marketdata"
	"github.com/alpacahq/alpaca-trade-api-go/v3/marketdata/stream"
)

type AlpacaProvider struct {
	client *stream.StocksClient
}

func NewAlpacaProvider(ctx context.Context, key, secret string) (*AlpacaProvider, error) {
	cli := stream.NewStocksClient(marketdata.IEX, stream.WithCredentials(key, secret))
	err := cli.Connect(ctx)
	if err != nil {
		return nil, err
	}

	return &AlpacaProvider{
		client: cli,
	}, nil
}

func (ap *AlpacaProvider) Name() string { return "alpaca" }

func (ap *AlpacaProvider) Subscribe(ctx context.Context, symbol string, tf string, handler barHandler) error {
	return ap.client.SubscribeToBars(func(b stream.Bar) {
		bar := domain.NewBar(b.Timestamp, symbol, tf, b.High, b.Open, b.Low, b.Close, b.Volume)
		handler(bar)
	}, symbol)
}
