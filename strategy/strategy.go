package strategy

import (
	"apex/internal/domain"
	"apex/strategy/internal/clients"
	"apex/strategy/internal/process"
	"apex/strategy/internal/storage"
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"os"

	"github.com/google/uuid"
)

var (
	ErrCantRollbackVersion error = errors.New("cannot rollback version")
	ErrStrategyIsntActive  error = errors.New("strategy is active")
)

type deployer interface {
	Deploy(context.Context, *domain.Strategy) (string, error)
	Teardown(context.Context, string) error
}

type store interface {
	Save(context.Context, *domain.Strategy) error
	Delete(context.Context, uuid.UUID) error
	Get(context.Context, uuid.UUID) (*domain.Strategy, error)
	List(context.Context) ([]*domain.Strategy, error)
}

type codeStore interface {
	Upload(context.Context, *domain.Strategy, []byte) (string, error)
	Download(context.Context, string) ([]byte, error)
	DownloadToDisk(context.Context, *domain.Strategy) (string, error)
}

type Service struct {
	deployer  deployer
	store     store
	fileStore codeStore
}

func New(db *sql.DB, log *slog.Logger) (*Service, error) {
	repo := storage.NewStore(db)
	client, err := clients.New(
		"localhost:9000",
		clients.WithCredentials(
			os.Getenv("MINIO_ROOT_USER"),
			os.Getenv("MINIO_ROOT_PASSWORD"),
		),
	)
	if err != nil {
		log.Error("failed to create minio client", "error", err)
		return nil, err
	}

	return &Service{
		store:     repo,
		fileStore: client,
		deployer:  process.New(),
	}, nil
}

func (s *Service) List(ctx context.Context) ([]*domain.Strategy, error) {
	return s.store.List(ctx)
}

func (s *Service) Create(ctx context.Context, name string, code []byte) (*domain.Strategy, error) {
	strategy := domain.NewStrategy(name)

	if _, err := s.fileStore.Upload(ctx, strategy, code); err != nil {
		return nil, err
	}

	if err := s.deploy(ctx, strategy); err != nil {
		return nil, err
	}

	return strategy, nil
}

func (s *Service) Upgrade(ctx context.Context, strategyId uuid.UUID, code []byte) error {
	strategy, err := s.store.Get(ctx, strategyId)
	if err != nil {
		return err
	}

	strategy.Version++

	if _, err := s.fileStore.Upload(ctx, strategy, code); err != nil {
		return err
	}

	if strategy.Status == domain.StatusActive {
		if err := s.deployer.Teardown(ctx, strategy.Identifier); err != nil {
			return err
		}
	}

	if err := s.deploy(ctx, strategy); err != nil {
		return err
	}

	return nil
}

func (s *Service) Start(ctx context.Context, strategyId uuid.UUID) error {
	strategy, err := s.store.Get(ctx, strategyId)
	if err != nil {
		return err
	}

	path, err := s.fileStore.DownloadToDisk(ctx, strategy)
	if err != nil {
		return err
	}
	strategy.Path = path

	if err := s.deploy(ctx, strategy); err != nil {
		return err
	}

	return nil
}

func (s *Service) deploy(ctx context.Context, strategy *domain.Strategy) error {
	ide, err := s.deployer.Deploy(ctx, strategy)
	if err != nil {
		return err
	}
	strategy.Identifier = ide

	strategy.Status = domain.StatusActive
	if err := s.store.Save(ctx, strategy); err != nil {
		return err
	}

	return nil
}

func (s *Service) Stop(ctx context.Context, strategyId uuid.UUID) error {
	strategy, err := s.store.Get(ctx, strategyId)
	if err != nil {
		return err
	}
	if strategy.Status != domain.StatusActive || strategy.Version <= 0 {
		return ErrStrategyIsntActive
	}

	if err := s.deployer.Teardown(ctx, strategy.Identifier); err != nil {
		return err
	}

	if err := s.store.Save(ctx, strategy); err != nil {
		return err
	}

	return nil
}

func (s *Service) Rollback(ctx context.Context, strategyId uuid.UUID) error {
	strategy, err := s.store.Get(ctx, strategyId)
	if err != nil {
		return err
	}

	if strategy.Version <= 1 {
		return ErrCantRollbackVersion
	}

	if err := s.deployer.Teardown(ctx, strategy.Identifier); err != nil {
		return err
	}
	strategy.Version--
	if _, err := s.deployer.Deploy(ctx, strategy); err != nil {
		return err
	}

	return nil
}

func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	strategy, err := s.store.Get(ctx, id)
	if err != nil {
		return err
	}

	if strategy.Status == domain.StatusActive {
		if err := s.deployer.Teardown(ctx, strategy.Identifier); err != nil {
			return err
		}
	}

	if err := s.store.Delete(ctx, id); err != nil {
		return err
	}

	return nil
}
