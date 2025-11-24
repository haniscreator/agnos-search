package repository

import (
	"context"
	"testing"
	"time"

	"github.com/pashagolub/pgxmock/v2"
	"github.com/stretchr/testify/assert"
)

func TestSearchPatients_ByNationalID(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	hid := "HIS-1"
	nid := "N-123"

	// Expect count query
	mock.ExpectQuery(`SELECT COUNT\(1\) FROM patients WHERE hospital_id = \$1 AND national_id = \$2`).
		WithArgs(hid, nid).
		WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(1))

	// Prepare row for select
	now := time.Now()
	rows := pgxmock.NewRows([]string{
		"id", "patient_hn", "national_id", "passport_id",
		"first_name_th", "middle_name_th", "last_name_th",
		"first_name_en", "middle_name_en", "last_name_en",
		"date_of_birth", "phone_number", "email", "gender", "raw_json",
	}).AddRow(
		"p1", "HN-1", nid, "P-1",
		"สมชาย", "", "ใจดี",
		"Somchai", "", "Jaidee",
		now, "0812345678", "a@example.com", "M", []byte(`{}`),
	)

	// Expect select query: note appended limit/offset arguments (we use limit=10 offset=0)
	mock.ExpectQuery(`SELECT id, patient_hn, national_id, passport_id, first_name_th`).
		WithArgs(hid, nid, 10, 0).
		WillReturnRows(rows)

	repo := NewPatientRepo(mock)
	results, total, err := repo.SearchPatients(context.Background(), hid, PatientFilters{NationalID: nid}, 10, 0)
	assert.NoError(t, err)
	assert.Equal(t, 1, total)
	if assert.Len(t, results, 1) {
		assert.Equal(t, "p1", results[0].ID)
		assert.Equal(t, nid, results[0].NationalID)
	}
	assert.NoError(t, mock.ExpectationsWereMet())
}
