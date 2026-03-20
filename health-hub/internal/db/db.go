package db

import (
	"context"
	_ "embed"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/001_init.sql
var migrationSQL string

func Connect(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse dsn: %w", err)
	}
	config.MaxConns = 10

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping: %w", err)
	}

	slog.Info("connected to database")
	return pool, nil
}

func RunMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	slog.Info("running migrations")
	_, err := pool.Exec(ctx, migrationSQL)
	if err != nil {
		return fmt.Errorf("migration: %w", err)
	}
	slog.Info("migrations complete")
	return nil
}
