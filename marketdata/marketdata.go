package marketdata

import (
	"apex/internal/domain"
	"apex/marketdata/internal/providers"
	"apex/marketdata/internal/storage"
	"context"
	"database/sql"
	"time"
)

/*
 * this file defines the public api of the marketdata package
 */

type marketDataStorage interface {
	StoreBar(context.Context, domain.Bar) error
	TrackSymbol(context.Context, string) error
	UntrackSymbol(context.Context, string) error
}

type Module struct {
	barStorage marketDataStorage
	provider   providers.Provider
}

func New(ctx context.Context, db *sql.DB, key, secret string) (*Module, error) {
	provider, err := providers.NewAlpacaProvider(ctx, key, secret)
	if err != nil {
		return nil, err
	}

	return &Module{
		barStorage: storage.NewRepo(db),
		provider:   provider,
	}, nil
}

func (m *Module) Subscribe(ctx context.Context, symbol, tf string) error {
	err := m.provider.Subscribe(ctx, symbol, tf, func(bar domain.Bar) error {
		return m.barStorage.StoreBar(ctx, bar)
	})
	if err != nil {
		return err
	}

	return nil
}

func (m *Module) Unsubscribe(ctx context.Context, symbol, tf string) error {
	return m.provider.Unsubscribe(ctx, symbol, tf)
}

func (m *Module) Backfill(ctx context.Context, symbol, tf string, start, end time.Time) error {
	bars, err := m.provider.GetBackfillBars(ctx, symbol, tf, start, end)
	if err != nil {
		return err
	}

	for _, bar := range bars {
		if err := m.barStorage.StoreBar(ctx, bar); err != nil {
			return err
		}
	}

	return nil
}
