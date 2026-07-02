package storage

import (
	"apex/internal/domain"
	"context"
	"database/sql"
)

type BarRepository struct {
	db *sql.DB
}

func NewRepo(db *sql.DB) *BarRepository {
	return &BarRepository{
		db: db,
	}
}

func (br *BarRepository) Store(ctx context.Context, bar domain.Bar) error {
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

func (br *BarRepository) List(ctx context.Context, symbol, tf string) ([]domain.Bar, error) {
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
