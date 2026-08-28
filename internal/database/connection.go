package database

import (
	"context"
	"fmt"

	"github.com/SOAT-15-Oficina/oficina-mecanica-monolith/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
)

const defaultMaxConnectionsPerReplica int32 = 5

func NewConnection(cfg *config.DatabaseConfig) (*pgxpool.Pool, error) {
	poolConfig, err := buildPoolConfig(cfg)
	if err != nil {
		return nil, err
	}

	pool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	if err := pool.Ping(context.Background()); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return pool, nil
}

func buildPoolConfig(cfg *config.DatabaseConfig) (*pgxpool.Config, error) {
	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Name,
	)

	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to parse connection pool configuration: %w", err)
	}

	maxConnections := cfg.MaxConnections
	if maxConnections <= 0 {
		maxConnections = defaultMaxConnectionsPerReplica
	}
	poolConfig.MaxConns = maxConnections

	return poolConfig, nil
}
