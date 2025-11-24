package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
)

// AnalyticsRepo defines audit logging operations used by handlers.
type AnalyticsRepo interface {
	// LogSearch records a search event (staff + hospital + filters + resultCount).
	LogSearch(ctx context.Context, staffID, hospitalID string, filters PatientFilters, resultCount int) error
}

// analyticsRepo is a simple Postgres-backed implementation.
type analyticsRepo struct {
	pool DBPool
}

func NewAnalyticsRepo(pool DBPool) AnalyticsRepo {
	return &analyticsRepo{pool: pool}
}

func (a *analyticsRepo) LogSearch(ctx context.Context, staffID, hospitalID string, filters PatientFilters, resultCount int) error {
	// serialize filters as JSONB
	b, err := json.Marshal(filters)
	if err != nil {
		return fmt.Errorf("marshal filters: %w", err)
	}
	_, err = a.pool.Exec(ctx,
		`INSERT INTO search_events (staff_id, hospital_id, filters, result_count) VALUES ($1,$2,$3,$4)`,
		staffID, hospitalID, b, resultCount,
	)
	var pgErr *pgconn.PgError
	if err != nil {
		// don't wrap too deeply here; return error as-is
		if ok := (err == pgErr); ok {
			return err
		}
		return err
	}
	return nil
}
