package storage

import (
	"apex/internal/domain"
	"apex/strategy/internal/storage/sqlcgen"
	"context"
	"database/sql"

	"github.com/google/uuid"
)

type Store struct {
	queries *sqlcgen.Queries
}

func NewStore(db *sql.DB) *Store {
	return &Store{queries: sqlcgen.New(db)}
}

func (s *Store) Save(ctx context.Context, strategy *domain.Strategy) error {
	return s.queries.UpsertStrategy(ctx, sqlcgen.UpsertStrategyParams{
		ID:         strategy.ID,
		Name:       strategy.Name,
		Status:     string(strategy.Status),
		Version:    int64(strategy.Version),
		Identifier: strategy.Identifier,
	})
}

func (s *Store) Get(ctx context.Context, id uuid.UUID) (*domain.Strategy, error) {
	row, err := s.queries.GetStrategy(ctx, id)
	if err != nil {
		return nil, err
	}
	return &domain.Strategy{
		ID:         row.ID,
		Name:       row.Name,
		Status:     domain.Status(row.Status),
		Version:    uint64(row.Version),
		Identifier: row.Identifier,
	}, nil
}

func (s *Store) Delete(ctx context.Context, id uuid.UUID) error {
	return s.queries.DeleteStrategy(ctx, id)
}

func (s *Store) List(ctx context.Context) ([]*domain.Strategy, error) {
	rows, err := s.queries.ListStrategies(ctx)
	if err != nil {
		return nil, err
	}

	strategies := make([]*domain.Strategy, len(rows))
	for i, row := range rows {
		strategies[i] = &domain.Strategy{
			ID:         row.ID,
			Name:       row.Name,
			Status:     domain.Status(row.Status),
			Version:    uint64(row.Version),
			Identifier: row.Identifier,
		}
	}
	return strategies, nil
}
