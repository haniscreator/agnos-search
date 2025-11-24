package repository

import (
	"context"
	"testing"
	"time"

	"github.com/pashagolub/pgxmock/v2"
	"github.com/stretchr/testify/assert"
)

func TestStaff_CreateAndGet(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	// Test Create expectation
	mock.ExpectExec(`INSERT INTO staffs \(`).
		WithArgs("s1", "alice", "hashedpw", "HIS-1", "Alice", "staff").
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	repo := NewStaffRepo(mock)
	ctx := context.Background()
	err = repo.Create(ctx, &Staff{
		ID:           "s1",
		Username:     "alice",
		PasswordHash: "hashedpw",
		HospitalID:   "HIS-1",
		DisplayName:  "Alice",
		Role:         "staff",
	})
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())

	// prepare time values for scan
	now := time.Now()

	rows := pgxmock.NewRows([]string{
		"id", "username", "password_hash", "hospital_id", "display_name", "role", "created_at", "updated_at",
	}).AddRow("s1", "alice", "hashedpw", "HIS-1", "Alice", "staff", now, now)

	mock.ExpectQuery(`SELECT id, username, password_hash, hospital_id, display_name, role, created_at, updated_at`).
		WithArgs("alice", "HIS-1").
		WillReturnRows(rows)

	got, err := repo.GetByUsernameAndHospital(ctx, "alice", "HIS-1")
	assert.NoError(t, err)
	if assert.NotNil(t, got) {
		assert.Equal(t, "s1", got.ID)
		assert.Equal(t, "alice", got.Username)
		assert.Equal(t, "HIS-1", got.HospitalID)
	}
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestStaff_Get_NotFound(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	mock.ExpectQuery(`SELECT id, username, password_hash, hospital_id, display_name, role, created_at, updated_at`).
		WithArgs("missing", "HIS-1").
		WillReturnRows(pgxmock.NewRows([]string{
			"id", "username", "password_hash", "hospital_id", "display_name", "role", "created_at", "updated_at",
		}))

	repo := NewStaffRepo(mock)
	ctx := context.Background()
	got, err := repo.GetByUsernameAndHospital(ctx, "missing", "HIS-1")
	assert.NoError(t, err)
	assert.Nil(t, got)
	assert.NoError(t, mock.ExpectationsWereMet())
}
