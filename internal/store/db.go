package store

import (
	"context"
	"fmt"

	"github.com/dorik33/cloud/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	pool             *pgxpool.Pool
	config           *config.Config
	ClientRepository ClientRepository
}

func NewConnection(cfg *config.Config) (*Store, error) {
	pool, err := pgxpool.New(context.Background(), cfg.DBConnStr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	if err := pool.Ping(context.Background()); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	store := &Store{
		pool:   pool,
		config: cfg,
	}

	store.ClientRepository = &clientRepository{store: store}

	return store, nil
}

func (s *Store) Close() {
	s.pool.Close()
}
