package storage

import (
	"apex/internal/domain"
	"context"
	"database/sql"
	"fmt"
	"strings"
)

type MarketDataRepository struct {
	db *sql.DB
}

func NewRepo(db *sql.DB) *MarketDataRepository {
	return &MarketDataRepository{
		db: db,
	}
}

func (br *MarketDataRepository) StoreBar(ctx context.Context, bar domain.Bar) error {
	_, err := br.db.ExecContext(
		ctx,
		`INSERT INTO bars (time, symbol, high, open, low, close, volume)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 ON CONFLICT (symbol, time) DO UPDATE SET
		     high = excluded.high,
		     open = excluded.open,
		     low = excluded.low,
		     close = excluded.close,
		     volume = excluded.volume`,
		bar.Time,
		bar.Symbol,
		bar.High,
		bar.Open,
		bar.Low,
		bar.Close,
		bar.Volume,
	)
	if err != nil {
		return err
	}

	return nil
}

// List returns stored 1-minute bars for symbol in chronological order.
func (br *MarketDataRepository) List(ctx context.Context, symbol string) ([]domain.Bar, error) {
	bars := make([]domain.Bar, 0)
	rows, err := br.db.QueryContext(ctx,
		`SELECT time, symbol, open, high, low, close, volume
		 FROM bars WHERE symbol = $1 ORDER BY time ASC`,
		symbol)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		bar := domain.Bar{}

		if err := rows.Scan(&bar.Time, &bar.Symbol, &bar.Open, &bar.High, &bar.Low, &bar.Close, &bar.Volume); err != nil {
			return nil, err
		}

		bars = append(bars, bar)
	}

	return bars, rows.Err()
}

// ListAggregated returns bars for symbol bucketed into the given TimescaleDB
// interval (e.g. "5 minutes", "1 hour"), computed on the fly via time_bucket
// over the stored 1-minute bars. bucket must come from a whitelisted set
// (see marketdata.intervalBuckets) - it is bound as a query parameter here,
// never string-concatenated, so an unexpected value fails to parse as an
// interval rather than becoming a SQL injection vector.
func (br *MarketDataRepository) ListAggregated(ctx context.Context, symbol, bucket string) ([]domain.Bar, error) {
	bars := make([]domain.Bar, 0)
	rows, err := br.db.QueryContext(ctx,
		`SELECT time_bucket($1::interval, time) AS bucket,
		        symbol,
		        first(open, time) AS open,
		        max(high) AS high,
		        min(low) AS low,
		        last(close, time) AS close,
		        sum(volume) AS volume
		 FROM bars
		 WHERE symbol = $2
		 GROUP BY bucket, symbol
		 ORDER BY bucket ASC`,
		bucket, symbol)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		bar := domain.Bar{}

		if err := rows.Scan(&bar.Time, &bar.Symbol, &bar.Open, &bar.High, &bar.Low, &bar.Close, &bar.Volume); err != nil {
			return nil, err
		}

		bars = append(bars, bar)
	}

	return bars, rows.Err()
}

const barsBulkInsertChunkSize = 500

// StoreBars upserts bars in chunked, multi-row INSERT statements so a large
// backfill (e.g. a year of 1-minute bars, ~98k rows) doesn't pay one
// round-trip per row.
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

	var sb strings.Builder
	sb.WriteString(`INSERT INTO bars (time, symbol, high, open, low, close, volume) VALUES `)
	args := make([]any, 0, len(chunk)*7)
	for i, bar := range chunk {
		if i > 0 {
			sb.WriteString(", ")
		}
		base := i * 7
		fmt.Fprintf(&sb, "($%d, $%d, $%d, $%d, $%d, $%d, $%d)",
			base+1, base+2, base+3, base+4, base+5, base+6, base+7)
		args = append(args, bar.Time, bar.Symbol, bar.High, bar.Open, bar.Low, bar.Close, bar.Volume)
	}
	sb.WriteString(` ON CONFLICT (symbol, time) DO UPDATE SET
		high = excluded.high,
		open = excluded.open,
		low = excluded.low,
		close = excluded.close,
		volume = excluded.volume`)

	_, err := br.db.ExecContext(ctx, sb.String(), args...)
	return err
}

func (br *MarketDataRepository) SaveSubscription(ctx context.Context, symbol string) error {
	_, err := br.db.ExecContext(ctx,
		`INSERT INTO subscriptions (symbol) VALUES ($1)
		 ON CONFLICT (symbol) DO NOTHING`,
		symbol)
	return err
}

func (br *MarketDataRepository) DeleteSubscription(ctx context.Context, symbol string) error {
	_, err := br.db.ExecContext(ctx,
		`DELETE FROM subscriptions WHERE symbol = $1`,
		symbol)
	return err
}

func (br *MarketDataRepository) ListSubscriptions(ctx context.Context) ([]domain.Subscription, error) {
	subs := make([]domain.Subscription, 0)
	rows, err := br.db.QueryContext(ctx,
		`SELECT symbol FROM subscriptions ORDER BY symbol`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var s domain.Subscription
		if err := rows.Scan(&s.Symbol); err != nil {
			return nil, err
		}
		subs = append(subs, s)
	}
	return subs, rows.Err()
}
