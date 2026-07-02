package marketdata

import (
	"apex/internal/domain"
	"apex/marketdata/internal/storage"
	"apex/marketdata/providers"
	"context"
	"database/sql"
)

/*
 * this file defines the public api of the marketdata package
 */

type BarStorage interface {
	Store(context.Context, domain.Bar) error
}

type Module struct {
	barStorage BarStorage
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
		return m.barStorage.Store(ctx, bar)
	})
	if err != nil {
		return err
	}

	return nil
}
