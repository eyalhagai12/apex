package storage

import (
	"apex/internal/domain"
	"context"
	"database/sql"

	"github.com/google/uuid"
)

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

func (s *Store) Save(ctx context.Context, strategy *domain.Strategy) error {
	query := `INSERT INTO strategies (id, name, status, version, identifier) VALUES ($1, $2, $3, $4, $5) ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, status = EXCLUDED.status, version = EXCLUDED.version, identifier = EXCLUDED.identifier`
	_, err := s.db.ExecContext(ctx, query, strategy.ID, strategy.Name, strategy.Status, strategy.Version, strategy.Identifier)
	return err
}

func (s *Store) Get(ctx context.Context, id uuid.UUID) (*domain.Strategy, error) {
	query := `SELECT id, name, status, version, identifier FROM strategies WHERE id = $1`
	row := s.db.QueryRowContext(ctx, query, id)
	var strategy domain.Strategy
	if err := row.Scan(&strategy.ID, &strategy.Name, &strategy.Status, &strategy.Version, &strategy.Identifier); err != nil {
		return nil, err
	}
	return &strategy, nil
}

func (s *Store) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM strategies WHERE id = $1`
	_, err := s.db.ExecContext(ctx, query, id)
	return err
}

func (s *Store) List(ctx context.Context) ([]*domain.Strategy, error) {
	query := `SELECT id, name, status, version, identifier FROM strategies ORDER BY name`
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	strategies := make([]*domain.Strategy, 0)
	for rows.Next() {
		var strategy domain.Strategy
		if err := rows.Scan(&strategy.ID, &strategy.Name, &strategy.Status, &strategy.Version, &strategy.Identifier); err != nil {
			return nil, err
		}
		strategies = append(strategies, &strategy)
	}
	return strategies, rows.Err()
}
