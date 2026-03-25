package db

import (
	"context"
	_ "embed"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/001_init.sql
var migration001 string

//go:embed migrations/002_unique_dedup.sql
var migration002 string

//go:embed migrations/003_nutrition_fields.sql
var migration003 string

//go:embed migrations/004_health_notes.sql
var migration004 string

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
	migrations := []struct {
		name string
		sql  string
	}{
		{"001_init", migration001},
		{"002_unique_dedup", migration002},
		{"003_nutrition_fields", migration003},
		{"004_health_notes", migration004},
	}

	for _, m := range migrations {
		slog.Info("running migration", "name", m.name)
		if _, err := pool.Exec(ctx, m.sql); err != nil {
			return fmt.Errorf("migration %s: %w", m.name, err)
		}
	}

	slog.Info("migrations complete")
	return nil
}
