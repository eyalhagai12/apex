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
		`INSERT INTO bars (time, symbol, timeframe, high, open, low, close, volume)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		 ON CONFLICT (symbol, timeframe, time) DO UPDATE SET
		     high = excluded.high,
		     open = excluded.open,
		     low = excluded.low,
		     close = excluded.close,
		     volume = excluded.volume`,
		bar.Time,
		bar.Symbol,
		bar.Timeframe,
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

func (br *MarketDataRepository) List(ctx context.Context, symbol, tf string) ([]domain.Bar, error) {
	bars := make([]domain.Bar, 0)
	rows, err := br.db.QueryContext(ctx,
		`SELECT time, symbol, timeframe, open, high, low, close, volume
		 FROM bars WHERE symbol = $1 AND timeframe = $2 ORDER BY time ASC`,
		symbol, tf)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		bar := domain.Bar{}

		if err := rows.Scan(&bar.Time, &bar.Symbol, &bar.Timeframe, &bar.Open, &bar.High, &bar.Low, &bar.Close, &bar.Volume); err != nil {
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
	sb.WriteString(`INSERT INTO bars (time, symbol, timeframe, high, open, low, close, volume) VALUES `)
	args := make([]any, 0, len(chunk)*8)
	for i, bar := range chunk {
		if i > 0 {
			sb.WriteString(", ")
		}
		base := i * 8
		fmt.Fprintf(&sb, "($%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d)",
			base+1, base+2, base+3, base+4, base+5, base+6, base+7, base+8)
		args = append(args, bar.Time, bar.Symbol, bar.Timeframe, bar.High, bar.Open, bar.Low, bar.Close, bar.Volume)
	}
	sb.WriteString(` ON CONFLICT (symbol, timeframe, time) DO UPDATE SET
		high = excluded.high,
		open = excluded.open,
		low = excluded.low,
		close = excluded.close,
		volume = excluded.volume`)

	_, err := br.db.ExecContext(ctx, sb.String(), args...)
	return err
}

func (mdr *MarketDataRepository) TrackSymbol(ctx context.Context, symbol string) error {
	row := mdr.db.QueryRowContext(ctx, "SELET id, name FROM symbols WHERE name = $1", symbol)

	var s string
	if err := row.Scan(&s); err == nil {
		return nil
	}

	_, err := mdr.db.ExecContext(ctx, "INSERT INTO symbols (name) VALUES ($1)", symbol)
	if err != nil {
		return err
	}

	return nil
}

func (mdr *MarketDataRepository) UntrackSymbol(ctx context.Context, symbol string) error {
	_, err := mdr.db.ExecContext(ctx, "DELETE FROM symbols WHERE name = $1", symbol)
	if err != nil {
		return err
	}

	return nil
}
