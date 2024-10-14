package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/stdlib"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mattiusz/based_backend/internal/config"
	"github.com/mattiusz/based_backend/internal/gen/sqlc"
	"github.com/pressly/goose/v3"
)

func NewDB(ctx context.Context, cfg *config.Config) (*pgxpool.Pool, error) {
	poolConfig, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database URL: %w", err)
	}

	// Set connection pool parameters
	poolConfig.MaxConns = 25
	poolConfig.MaxConnLifetime = 5 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Ping to verify connection
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return pool, nil
}

func RunMigrations(ctx context.Context, db *pgxpool.Pool) error {
	conn, err := db.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("failed to acquire connection for migrations: %w", err)
	}
	defer conn.Release()

	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}

	sqlDB := stdlib.OpenDBFromPool(conn)
	if err != nil {
		return fmt.Errorf("failed to get standard lib connection: %w", err)
	}

	return goose.Up(sqlDB, "./migrations")
}

func NewQueries(db *pgxpool.Pool) *sqlc.Queries {
	return sqlc.New(db)
}
