package repository

import (
	"context"
	"testing"
	"time"

	"github.com/pashagolub/pgxmock/v2"
	"github.com/stretchr/testify/assert"
)

func TestGetByID_Found(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	// prepare a time.Time value for date_of_birth (matches sql.NullTime scan)
	dob, err := time.Parse("2006-01-02", "1990-01-01")
	assert.NoError(t, err)

	// prepare rows matching the SELECT column order used in GetByID
	rows := pgxmock.NewRows([]string{
		"id", "patient_hn", "national_id", "passport_id",
		"first_name_th", "middle_name_th", "last_name_th",
		"first_name_en", "middle_name_en", "last_name_en",
		"date_of_birth", "phone_number", "email", "gender", "raw_json",
	}).AddRow(
		"p1", "HN-1", "N-123", "P-1",
		"สมชาย", "", "ใจดี",
		"Somchai", "", "Jaidee",
		dob, "0812345678", "a@example.com", "M", []byte(`{"source":"test"}`),
	)

	mock.ExpectQuery(`SELECT id, patient_hn, national_id, passport_id,`).
		WithArgs("p1").
		WillReturnRows(rows)

	repo := NewPatientRepo(mock)
	ctx := context.Background()
	p, err := repo.GetByID(ctx, "p1")
	assert.NoError(t, err)
	assert.NotNil(t, p)
	assert.Equal(t, "p1", p.ID)
	assert.Equal(t, "N-123", p.NationalID)
	assert.Equal(t, "Somchai", p.FirstNameEN)
	assert.Equal(t, "HN-1", p.PatientHN)
	if assert.NotNil(t, p.DateOfBirth) {
		assert.Equal(t, "1990-01-01", *p.DateOfBirth)
	}

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetByID_NotFound(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	mock.ExpectQuery(`SELECT id, patient_hn, national_id, passport_id,`).
		WithArgs("missing").
		WillReturnRows(pgxmock.NewRows([]string{
			"id", "patient_hn", "national_id", "passport_id",
			"first_name_th", "middle_name_th", "last_name_th",
			"first_name_en", "middle_name_en", "last_name_en",
			"date_of_birth", "phone_number", "email", "gender", "raw_json",
		})) // zero rows

	repo := NewPatientRepo(mock)
	ctx := context.Background()
	p, err := repo.GetByID(ctx, "missing")
	assert.NoError(t, err)
	assert.Nil(t, p)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCreate_Success(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	// Expect Exec for INSERT with 15 args. For date_of_birth we expect a typed nil (*string)(nil)
	mock.ExpectExec(`INSERT INTO patients \(`).
		WithArgs(
			"p2", "HN-2", "N-456", "P-2",
			"JaneTH", "MiddTH", "LastTH",
			"Jane", "MiddEN", "LastEN",
			(*string)(nil), // typed nil matches actual *string(nil) passed by repo.Create
			"0999", "jane@example.com", "F", []byte(`{"k":"v"}`),
		).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	repo := NewPatientRepo(mock)
	ctx := context.Background()
	err = repo.Create(ctx, &Patient{
		ID:           "p2",
		PatientHN:    "HN-2",
		NationalID:   "N-456",
		PassportID:   "P-2",
		FirstNameTH:  "JaneTH",
		MiddleNameTH: "MiddTH",
		LastNameTH:   "LastTH",
		FirstNameEN:  "Jane",
		MiddleNameEN: "MiddEN",
		LastNameEN:   "LastEN",
		DateOfBirth:  nil, // nil pointer of type *string
		PhoneNumber:  "0999",
		Email:        "jane@example.com",
		Gender:       "F",
		RawJSON:      []byte(`{"k":"v"}`),
	})
	assert.NoError(t, err)

	assert.NoError(t, mock.ExpectationsWereMet())
}

// helper reused in tests (kept for compatibility)
func strptr(s string) *string { return &s }
