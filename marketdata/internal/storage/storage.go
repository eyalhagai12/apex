package storage

import (
	"apex/internal/domain"
	"apex/marketdata/internal/storage/sqlcgen"
	"context"
	"database/sql"
	"fmt"
	"time"
)

type MarketDataRepository struct {
	queries *sqlcgen.Queries
}

func NewRepo(db *sql.DB) *MarketDataRepository {
	return &MarketDataRepository{
		queries: sqlcgen.New(db),
	}
}

func (br *MarketDataRepository) StoreBar(ctx context.Context, bar domain.Bar) error {
	return br.queries.UpsertBar(ctx, sqlcgen.UpsertBarParams{
		Time:   bar.Time,
		Symbol: bar.Symbol,
		High:   bar.High,
		Open:   bar.Open,
		Low:    bar.Low,
		Close:  bar.Close,
		Volume: int64(bar.Volume),
	})
}

// List returns stored 1-minute bars for symbol in chronological order.
func (br *MarketDataRepository) List(ctx context.Context, symbol string) ([]domain.Bar, error) {
	rows, err := br.queries.ListBarsBySymbol(ctx, symbol)
	if err != nil {
		return nil, err
	}

	bars := make([]domain.Bar, len(rows))
	for i, row := range rows {
		bars[i] = domain.Bar{
			Time:   row.Time,
			Symbol: row.Symbol,
			High:   row.High,
			Open:   row.Open,
			Low:    row.Low,
			Close:  row.Close,
			Volume: uint64(row.Volume),
		}
	}
	return bars, nil
}

// ListAggregated returns bars for symbol bucketed into the given TimescaleDB
// interval (e.g. "5 minutes", "1 hour"), computed on the fly via time_bucket
// over the stored 1-minute bars. bucket must come from a whitelisted set
// (see marketdata.intervalBuckets) - it is bound as a query parameter here,
// never string-concatenated, so an unexpected value fails to parse as an
// interval rather than becoming a SQL injection vector.
func (br *MarketDataRepository) ListAggregated(ctx context.Context, symbol, bucket string) ([]domain.Bar, error) {
	rows, err := br.queries.ListAggregatedBars(ctx, sqlcgen.ListAggregatedBarsParams{
		Bucket: bucket,
		Symbol: symbol,
	})
	if err != nil {
		return nil, err
	}

	bars := make([]domain.Bar, len(rows))
	for i, row := range rows {
		bars[i] = domain.Bar{
			Time:   row.Bucket,
			Symbol: row.Symbol,
			High:   row.High,
			Open:   row.Open,
			Low:    row.Low,
			Close:  row.Close,
			Volume: uint64(row.Volume),
		}
	}
	return bars, nil
}

const barsBulkInsertChunkSize = 500

// StoreBars upserts bars in chunked, set-based upserts (via unnest over
// parallel arrays, see UpsertBars) so a large backfill (e.g. a year of
// 1-minute bars, ~98k rows) doesn't pay one round-trip per row.
func (br *MarketDataRepository) StoreBars(ctx context.Context, bars []domain.Bar) error {
	for start := 0; start < len(bars); start += barsBulkInsertChunkSize {
		end := start + barsBulkInsertChunkSize
		if end > len(bars) {
			end = len(bars)
		}
		if err := br.storeBarsChunk(ctx, bars[start:end]); err != nil {
			return fmt.Errorf("store bars chunk [%d:%d]: %w", start, end, err)
		}
	}
	return nil
}

func (br *MarketDataRepository) storeBarsChunk(ctx context.Context, chunk []domain.Bar) error {
	if len(chunk) == 0 {
		return nil
	}

	params := sqlcgen.UpsertBarsParams{
		Time:   make([]time.Time, len(chunk)),
		Symbol: make([]string, len(chunk)),
		High:   make([]float64, len(chunk)),
		Open:   make([]float64, len(chunk)),
		Low:    make([]float64, len(chunk)),
		Close:  make([]float64, len(chunk)),
		Volume: make([]int64, len(chunk)),
	}
	for i, bar := range chunk {
		params.Time[i] = bar.Time
		params.Symbol[i] = bar.Symbol
		params.High[i] = bar.High
		params.Open[i] = bar.Open
		params.Low[i] = bar.Low
		params.Close[i] = bar.Close
		params.Volume[i] = int64(bar.Volume)
	}

	return br.queries.UpsertBars(ctx, params)
}

func (br *MarketDataRepository) SaveSubscription(ctx context.Context, symbol string) error {
	return br.queries.UpsertSubscription(ctx, symbol)
}

func (br *MarketDataRepository) DeleteSubscription(ctx context.Context, symbol string) error {
	return br.queries.DeleteSubscription(ctx, symbol)
}

func (br *MarketDataRepository) ListSubscriptions(ctx context.Context) ([]domain.Subscription, error) {
	symbols, err := br.queries.ListSubscriptions(ctx)
	if err != nil {
		return nil, err
	}

	subs := make([]domain.Subscription, len(symbols))
	for i, symbol := range symbols {
		subs[i] = domain.Subscription{Symbol: symbol}
	}
	return subs, nil
}
