package storage

import (
	"apex/internal/domain"
	"context"
	"database/sql"
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
	query, err := br.db.QueryContext(ctx, "SELECT * FROM bars WHERE symbol = $1 AND timeframe = $2", symbol, tf)
	if err != nil {
		return nil, err
	}

	for query.Next() {
		bar := domain.Bar{}

		if err := query.Scan(&bar.Time, &bar.Symbol, &bar.Timeframe, &bar.High, &bar.Open, &bar.Low, &bar.Close, &bar.Volume); err != nil {
			return nil, err
		}

		bars = append(bars, bar)
	}

	return bars, nil
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
