package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Patient represents the normalized patient model stored in DB.
type Patient struct {
	ID           string
	PatientHN    string
	NationalID   string
	PassportID   string
	FirstNameTH  string
	MiddleNameTH string
	LastNameTH   string
	FirstNameEN  string
	MiddleNameEN string
	LastNameEN   string
	DateOfBirth  *string // yyyy-mm-dd string; nil if absent
	PhoneNumber  string
	Email        string
	Gender       string // 'M' or 'F'
	RawJSON      []byte // optional raw JSON
}

// DBPool is a minimal subset of pgxpool.Pool used by the repo.
type DBPool interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// PatientRepo handles patient persistence
type PatientRepo struct {
	pool DBPool
}

func NewPatientRepo(pool DBPool) *PatientRepo {
	return &PatientRepo{pool: pool}
}

// GetByID fetches a patient by internal UUID id.
// Returns (nil, nil) if not found.
func (r *PatientRepo) GetByID(ctx context.Context, id string) (*Patient, error) {
	row := r.pool.QueryRow(ctx, `
    SELECT id, patient_hn, national_id, passport_id,
           first_name_th, middle_name_th, last_name_th,
           first_name_en, middle_name_en, last_name_en,
           date_of_birth, phone_number, email, gender, raw_json
    FROM patients WHERE id = $1`, id)

	var p Patient
	var dob sql.NullTime
	var raw []byte

	err := row.Scan(
		&p.ID,
		&p.PatientHN,
		&p.NationalID,
		&p.PassportID,
		&p.FirstNameTH,
		&p.MiddleNameTH,
		&p.LastNameTH,
		&p.FirstNameEN,
		&p.MiddleNameEN,
		&p.LastNameEN,
		&dob,
		&p.PhoneNumber,
		&p.Email,
		&p.Gender,
		&raw,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	if dob.Valid {
		str := dob.Time.Format("2006-01-02")
		p.DateOfBirth = &str
	}
	p.RawJSON = raw
	return &p, nil
}

// GetByIdentifier finds a patient by national_id OR passport_id (input can be either).
// Returns (nil, nil) if not found.
func (r *PatientRepo) GetByIdentifier(ctx context.Context, identifier string) (*Patient, error) {
	row := r.pool.QueryRow(ctx, `
    SELECT id, patient_hn, national_id, passport_id,
           first_name_th, middle_name_th, last_name_th,
           first_name_en, middle_name_en, last_name_en,
           date_of_birth, phone_number, email, gender, raw_json
    FROM patients WHERE national_id = $1 OR passport_id = $1 LIMIT 1`, identifier)

	var p Patient
	var dob sql.NullTime
	var raw []byte

	err := row.Scan(
		&p.ID,
		&p.PatientHN,
		&p.NationalID,
		&p.PassportID,
		&p.FirstNameTH,
		&p.MiddleNameTH,
		&p.LastNameTH,
		&p.FirstNameEN,
		&p.MiddleNameEN,
		&p.LastNameEN,
		&dob,
		&p.PhoneNumber,
		&p.Email,
		&p.Gender,
		&raw,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	if dob.Valid {
		str := dob.Time.Format("2006-01-02")
		p.DateOfBirth = &str
	}
	p.RawJSON = raw
	return &p, nil
}

// Create inserts a new patient
func (r *PatientRepo) Create(ctx context.Context, p *Patient) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO patients (
			id, patient_hn, national_id, passport_id,
			first_name_th, middle_name_th, last_name_th,
			first_name_en, middle_name_en, last_name_en,
			date_of_birth, phone_number, email, gender, raw_json
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)`,
		p.ID, p.PatientHN, p.NationalID, p.PassportID,
		p.FirstNameTH, p.MiddleNameTH, p.LastNameTH,
		p.FirstNameEN, p.MiddleNameEN, p.LastNameEN,
		p.DateOfBirth, p.PhoneNumber, p.Email, p.Gender, p.RawJSON)
	return err
}
