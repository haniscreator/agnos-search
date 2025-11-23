package db

import (
	"context"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// NewPool creates a pgxpool.Pool from DATABASE_URL env var.
// It is used by the running app; tests will use pgxmock instead.
func NewPool(ctx context.Context) (*pgxpool.Pool, error) {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		// fallback for local dev if you want to run real DB
		url = "postgres://agnos:secret@localhost:5432/agnos?sslmode=disable"
	}

	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, err
	}
	// sensible defaults
	cfg.MaxConns = 5
	cfg.HealthCheckPeriod = 30 * time.Second

	return pgxpool.NewWithConfig(ctx, cfg)
}
