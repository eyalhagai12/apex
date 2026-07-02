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
		 ON CONFLICT (symbol, timeframe, time) DO NOTHING`,
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
