package db

import (
	"context"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mattiusz/based_backend/internal/config"
)

func NewDB(ctx context.Context, cfg *config.Config) (*pgxpool.Pool, error) {
	// Construct database URL for the migrate instance
	dbURL := fmt.Sprintf("postgres://%s:%s@%s:%s/%s", cfg.DatabaseUser, cfg.DatabasePassword, cfg.DatabaseHost, cfg.DatabasePort, cfg.DatabaseName)

	poolConfig, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database URL: %w", err)
	}

	// Set connection pool parameters
	poolConfig.MaxConns = cfg.MaxPoolConns
	poolConfig.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		dataTypeNames := []string{
			// Custom sqlc data types
			"gender_type",
			"_gender_type", // array type
			"event_status_type",
			"_event_status_type", // array type
			"event_category_type",
			"_event_category_type", // array type
		}
		for _, typeName := range dataTypeNames {
			dataType, _ := conn.LoadType(ctx, typeName)
			conn.TypeMap().RegisterType(dataType)
		}
		return nil
	}

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

func RunMigrations(ctx context.Context, cfg *config.Config) error {
	// Construct database URL for the migrate instance
	dbURL := fmt.Sprintf("pgx://%s:%s@%s:%s/%s", cfg.DatabaseUser, cfg.DatabasePassword, cfg.DatabaseHost, cfg.DatabasePort, cfg.DatabaseName)

	// Initialize migrator with file source and database URL
	m, err := migrate.New(cfg.MigrationsDir, dbURL)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}
	defer func() {
		sourceErr, dbErr := m.Close()
		if sourceErr != nil {
			fmt.Printf("Error closing migration source: %v\n", sourceErr)
		}
		if dbErr != nil {
			fmt.Printf("Error closing migration database: %v\n", dbErr)
		}
	}()

	// Run migrations
	if err := m.Up(); err != nil {
		if err == migrate.ErrNoChange {
			// No migrations to run is not an error
			return nil
		}
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}
