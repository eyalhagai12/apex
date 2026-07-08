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
	StoreBars(context.Context, []domain.Bar) error
	List(context.Context, string, string) ([]domain.Bar, error)
	TrackSymbol(context.Context, string) error
	UntrackSymbol(context.Context, string) error
	SaveSubscription(context.Context, string, string) error
	DeleteSubscription(context.Context, string, string) error
	ListSubscriptions(context.Context) ([]domain.Subscription, error)
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

// Subscribe starts streaming live bars for symbol/tf, storing each to the DB.
// If onBar is non-nil, it is also invoked for every live bar received (e.g.
// so the web layer can fan bars out over SSE, or a future strategy consumer
// can attach to the same stream). The subscription is persisted before the
// live provider subscription is started, so a failed persist never leaves a
// live-but-unpersisted subscription behind.
func (m *Module) Subscribe(ctx context.Context, symbol, tf string, onBar func(domain.Bar)) error {
	if err := m.barStorage.SaveSubscription(ctx, symbol, tf); err != nil {
		return err
	}

	metrics.BarsStreamed.WithLabelValues(symbol, tf) // see comment in Backfill

	return m.provider.Subscribe(ctx, symbol, tf, func(bar domain.Bar) error {
		if err := m.barStorage.StoreBar(ctx, bar); err != nil {
			return err
		}
		metrics.BarsStreamed.WithLabelValues(symbol, tf).Inc()
		if onBar != nil {
			onBar(bar)
		}
		return nil
	})
}

func (m *Module) Unsubscribe(ctx context.Context, symbol, tf string) error {
	if err := m.provider.Unsubscribe(ctx, symbol, tf); err != nil {
		return err
	}
	return m.barStorage.DeleteSubscription(ctx, symbol, tf)
}

// ListSubscriptions returns every symbol/timeframe pair currently persisted
// as subscribed, e.g. to resume live streaming for each after a restart.
func (m *Module) ListSubscriptions(ctx context.Context) ([]domain.Subscription, error) {
	return m.barStorage.ListSubscriptions(ctx)
}

// BackfillResult describes the outcome of an async Backfill call.
type BackfillResult struct {
	Symbol    string
	Timeframe string
	NumBars   int
	Err       error
}

// Backfill fetches historical bars for symbol/tf between start and end and
// stores them, running asynchronously (the caller is not blocked). If
// onComplete is non-nil, it is invoked once the goroutine finishes with the
// outcome, since the returned error only reflects whether the goroutine was
// launched, not whether it succeeded.
func (m *Module) Backfill(ctx context.Context, symbol, tf string, start, end time.Time, onComplete func(BackfillResult)) error {
	// Touch the series now, before the async work below, so Prometheus can
	// scrape it at 0 first. A CounterVec label combination doesn't exist
	// until WithLabelValues is called - if that first call happened only
	// after Add() below, the series would appear already at its final
	// value and increase()/rate() would never see it rise.
	metrics.BarsBackfilled.WithLabelValues(symbol, tf)

	m.errgp.Go(func() error {
		bars, err := m.provider.GetBackfillBars(ctx, symbol, tf, start, end)
		if err != nil {
			if onComplete != nil {
				onComplete(BackfillResult{Symbol: symbol, Timeframe: tf, Err: err})
			}
			return err
		}
		if len(bars) <= 0 {
			m.log.Warn("no bars returned", slog.String("symbol", symbol), slog.String("tf", tf), slog.Time("start", start), slog.Time("end", end))
			err := errors.New("no bars returned")
			if onComplete != nil {
				onComplete(BackfillResult{Symbol: symbol, Timeframe: tf, Err: err})
			}
			return err
		}

		if err := m.barStorage.StoreBars(ctx, bars); err != nil {
			if onComplete != nil {
				onComplete(BackfillResult{Symbol: symbol, Timeframe: tf, Err: err})
			}
			return err
		}
		metrics.BarsBackfilled.WithLabelValues(symbol, tf).Add(float64(len(bars)))

		m.log.Info("backfill done", slog.String("symbol", symbol), slog.String("tf", tf), slog.Time("start", start), slog.Time("end", end), slog.Int("n_bars", len(bars)))

		if onComplete != nil {
			onComplete(BackfillResult{Symbol: symbol, Timeframe: tf, NumBars: len(bars)})
		}
		return nil
	})

	return nil
}

// ListBars returns stored bars for symbol/tf in chronological order.
func (m *Module) ListBars(ctx context.Context, symbol, tf string) ([]domain.Bar, error) {
	return m.barStorage.List(ctx, symbol, tf)
}
