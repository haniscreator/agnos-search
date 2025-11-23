package repository

import (
	"context"
	"testing"

	"github.com/pashagolub/pgxmock/v2"
	"github.com/stretchr/testify/assert"
)

func TestGetByID_Found(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	// expected SQL and return row
	rows := pgxmock.NewRows([]string{"id", "national_id", "name"}).
		AddRow("p1", "N-123", "John Doe")

	mock.ExpectQuery(`SELECT id, national_id, name FROM patients WHERE id = \$1`).
		WithArgs("p1").
		WillReturnRows(rows)

	repo := NewPatientRepo(mock)
	ctx := context.Background()
	p, err := repo.GetByID(ctx, "p1")
	assert.NoError(t, err)
	assert.NotNil(t, p)
	assert.Equal(t, "p1", p.ID)
	assert.Equal(t, "N-123", p.NationalID)
	assert.Equal(t, "John Doe", p.Name)

	// ensure expectations met
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetByID_NotFound(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	// return no rows -> pgxmock simulates no rows automatically when no row set
	mock.ExpectQuery(`SELECT id, national_id, name FROM patients WHERE id = \$1`).
		WithArgs("missing").
		WillReturnRows(pgxmock.NewRows([]string{"id", "national_id", "name"})) // zero rows

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

	mock.ExpectExec(`INSERT INTO patients \(id, national_id, name\) VALUES \(\$1, \$2, \$3\)`).
		WithArgs("p2", "N-456", "Jane").
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	repo := NewPatientRepo(mock)
	ctx := context.Background()
	err = repo.Create(ctx, &Patient{ID: "p2", NationalID: "N-456", Name: "Jane"})
	assert.NoError(t, err)

	assert.NoError(t, mock.ExpectationsWereMet())
}
