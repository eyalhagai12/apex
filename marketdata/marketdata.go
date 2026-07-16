package marketdata

import (
	"apex/internal/domain"
	"apex/internal/metrics"
	"apex/marketdata/internal/providers"
	"apex/marketdata/internal/storage"
	"context"
	"database/sql"
	"errors"
	"fmt"
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
	List(context.Context, string) ([]domain.Bar, error)
	ListAggregated(context.Context, string, string) ([]domain.Bar, error)
	SaveSubscription(context.Context, string) error
	DeleteSubscription(context.Context, string) error
	ListSubscriptions(context.Context) ([]domain.Subscription, error)
}

// intervalBuckets whitelists the display intervals ListBars may aggregate to
// beyond the raw 1-minute storage resolution, mapping each to a TimescaleDB
// interval literal used in time_bucket($1::interval, time).
var intervalBuckets = map[string]string{
	"5Min":  "5 minutes",
	"15Min": "15 minutes",
	"1H":    "1 hour",
	"1D":    "1 day",
}

// ErrUnsupportedInterval is returned by ListBars for an interval not in
// intervalBuckets, so callers (e.g. the web layer) can distinguish a bad
// request from a storage/DB failure.
var ErrUnsupportedInterval = errors.New("unsupported interval")

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
func (m *Module) Subscribe(ctx context.Context, symbol string, onBar func(domain.Bar)) error {
	if err := m.barStorage.SaveSubscription(ctx, symbol); err != nil {
		return err
	}

	metrics.BarsStreamed.WithLabelValues(symbol) // see comment in Backfill

	return m.provider.Subscribe(ctx, symbol, func(bar domain.Bar) error {
		if err := m.barStorage.StoreBar(ctx, bar); err != nil {
			return err
		}
		metrics.BarsStreamed.WithLabelValues(symbol).Inc()
		if onBar != nil {
			onBar(bar)
		}
		return nil
	})
}

func (m *Module) Unsubscribe(ctx context.Context, symbol string) error {
	if err := m.provider.Unsubscribe(ctx, symbol); err != nil {
		return err
	}
	return m.barStorage.DeleteSubscription(ctx, symbol)
}

// ListSubscriptions returns every symbol/timeframe pair currently persisted
// as subscribed, e.g. to resume live streaming for each after a restart.
func (m *Module) ListSubscriptions(ctx context.Context) ([]domain.Subscription, error) {
	return m.barStorage.ListSubscriptions(ctx)
}

// BackfillResult describes the outcome of an async Backfill call.
type BackfillResult struct {
	Symbol    string
	NumBars   int
	Err       error
}

// Backfill fetches historical bars for symbol/tf between start and end and
// stores them, running asynchronously (the caller is not blocked). If
// onComplete is non-nil, it is invoked once the goroutine finishes with the
// outcome, since the returned error only reflects whether the goroutine was
// launched, not whether it succeeded.
func (m *Module) Backfill(ctx context.Context, symbol string, start, end time.Time, onComplete func(BackfillResult)) error {
	// Touch the series now, before the async work below, so Prometheus can
	// scrape it at 0 first. A CounterVec label combination doesn't exist
	// until WithLabelValues is called - if that first call happened only
	// after Add() below, the series would appear already at its final
	// value and increase()/rate() would never see it rise.
	metrics.BarsBackfilled.WithLabelValues(symbol)

	m.errgp.Go(func() error {
		bars, err := m.provider.GetBackfillBars(ctx, symbol, start, end)
		if err != nil {
			if onComplete != nil {
				onComplete(BackfillResult{Symbol: symbol, Err: err})
			}
			return err
		}
		if len(bars) <= 0 {
			m.log.Warn("no bars returned", slog.String("symbol", symbol), slog.Time("start", start), slog.Time("end", end))
			err := errors.New("no bars returned")
			if onComplete != nil {
				onComplete(BackfillResult{Symbol: symbol, Err: err})
			}
			return err
		}

		if err := m.barStorage.StoreBars(ctx, bars); err != nil {
			if onComplete != nil {
				onComplete(BackfillResult{Symbol: symbol, Err: err})
			}
			return err
		}
		metrics.BarsBackfilled.WithLabelValues(symbol).Add(float64(len(bars)))

		m.log.Info("backfill done", slog.String("symbol", symbol), slog.Time("start", start), slog.Time("end", end), slog.Int("n_bars", len(bars)))

		if onComplete != nil {
			onComplete(BackfillResult{Symbol: symbol, NumBars: len(bars)})
		}
		return nil
	})

	return nil
}

// ListBars returns bars for symbol in chronological order, at the given
// display interval. "1Min" (or empty) returns the raw stored bars directly;
// any other interval must be in intervalBuckets and is computed on the fly
// via TimescaleDB time_bucket over the stored 1-minute data.
func (m *Module) ListBars(ctx context.Context, symbol, interval string) ([]domain.Bar, error) {
	if interval == "" || interval == "1Min" {
		return m.barStorage.List(ctx, symbol)
	}
	bucket, ok := intervalBuckets[interval]
	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrUnsupportedInterval, interval)
	}
	return m.barStorage.ListAggregated(ctx, symbol, bucket)
}
