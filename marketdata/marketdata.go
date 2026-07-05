package marketdata

import (
	"apex/internal/domain"
	"apex/internal/metrics"
	"apex/marketdata/internal/providers"
	"apex/marketdata/internal/storage"
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"time"

	"golang.org/x/sync/errgroup"
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

	log   *slog.Logger
	errgp *errgroup.Group
}

func New(ctx context.Context, db *sql.DB, log *slog.Logger, key, secret string) (*Module, error) {
	provider, err := providers.NewAlpacaProvider(ctx, key, secret)
	if err != nil {
		return nil, err
	}

	return &Module{
		barStorage: storage.NewRepo(db),
		provider:   provider,
		log:        log,
		errgp:      &errgroup.Group{},
	}, nil
}

func (m *Module) Subscribe(ctx context.Context, symbol, tf string) error {
	err := m.provider.Subscribe(ctx, symbol, tf, func(bar domain.Bar) error {
		if err := m.barStorage.StoreBar(ctx, bar); err != nil {
			return err
		}
		metrics.BarsStreamed.WithLabelValues(symbol, tf).Inc()
		return nil
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
	m.errgp.Go(func() error {
		bars, err := m.provider.GetBackfillBars(ctx, symbol, tf, start, end)
		if err != nil {
			return err
		}
		if len(bars) <= 0 {
			m.log.Warn("no bars returned", slog.String("symbol", symbol), slog.String("tf", tf), slog.Time("start", start), slog.Time("end", end))
			return errors.New("no bars returned")
		}

		for _, bar := range bars {
			if err := m.barStorage.StoreBar(ctx, bar); err != nil {
				return err
			}
			metrics.BarsBackfilled.WithLabelValues(symbol, tf).Inc()
		}

		m.log.Info("backfill done", slog.String("symbol", symbol), slog.String("tf", tf), slog.Time("start", start), slog.Time("end", end), slog.Int("n_bars", len(bars)))

		return nil
	})

	return nil
}
