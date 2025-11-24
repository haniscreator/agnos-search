package repository

import (
	"context"
	"testing"

	"github.com/pashagolub/pgxmock/v2"
	"github.com/stretchr/testify/assert"
)

func TestUpsert_ByNationalID(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	// expect Exec with ON CONFLICT (national_id)
	mock.ExpectExec(`ON CONFLICT \(national_id\)`).
		WithArgs(
			"p1", "HN-1", "N-123", "P-1",
			"สมชาย", "", "ใจดี",
			"Somchai", "", "Jaidee",
			(*string)(nil), // <- typed nil to match repo.Upsert argument
			"0812345678", "a@example.com", "M", []byte(`{}`), "HIS-1",
		).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	repo := NewPatientRepo(mock)
	p := &Patient{
		ID:           "p1",
		PatientHN:    "HN-1",
		NationalID:   "N-123",
		PassportID:   "P-1",
		FirstNameTH:  "สมชาย",
		MiddleNameTH: "",
		LastNameTH:   "ใจดี",
		FirstNameEN:  "Somchai",
		MiddleNameEN: "",
		LastNameEN:   "Jaidee",
		DateOfBirth:  nil,
		PhoneNumber:  "0812345678",
		Email:        "a@example.com",
		Gender:       "M",
		RawJSON:      []byte(`{}`),
		HospitalID:   "HIS-1",
	}

	err = repo.Upsert(context.Background(), p)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}
