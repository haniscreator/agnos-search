package repository

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

// Staff model stored in DB
type Staff struct {
	ID           string
	Username     string
	PasswordHash string
	HospitalID   string
	DisplayName  string
	Role         string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// StaffRepo handles staff persistence
type StaffRepo struct {
	pool DBPool // reuse DBPool type from repository package
}

func NewStaffRepo(pool DBPool) *StaffRepo {
	return &StaffRepo{pool: pool}
}

// Create inserts a new staff. Expects caller to generate ID and hash password.
func (r *StaffRepo) Create(ctx context.Context, s *Staff) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO staffs (id, username, password_hash, hospital_id, display_name, role)
		 VALUES ($1,$2,$3,$4,$5,$6)`,
		s.ID, s.Username, s.PasswordHash, s.HospitalID, s.DisplayName, s.Role,
	)
	return err
}

// GetByUsernameAndHospital returns staff by username and hospital.
// Returns (nil, nil) if not found.
func (r *StaffRepo) GetByUsernameAndHospital(ctx context.Context, username, hospitalID string) (*Staff, error) {
	row := r.pool.QueryRow(ctx, `
    SELECT id, username, password_hash, hospital_id, display_name, role, created_at, updated_at
    FROM staffs
    WHERE username = $1 AND hospital_id = $2
    LIMIT 1`, username, hospitalID)

	var s Staff
	err := row.Scan(
		&s.ID,
		&s.Username,
		&s.PasswordHash,
		&s.HospitalID,
		&s.DisplayName,
		&s.Role,
		&s.CreatedAt,
		&s.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &s, nil
}
