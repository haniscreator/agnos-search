package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

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
	HospitalID   string // which hospital the patient belongs to
}

// DBPool is a minimal subset of pgxpool.Pool used by the repo.
type DBPool interface {
	QueryRow(ctx context.Context, query string, args ...any) pgx.Row
	Query(ctx context.Context, query string, args ...any) (pgx.Rows, error)
	Exec(ctx context.Context, query string, args ...any) (pgconn.CommandTag, error)
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
       date_of_birth, phone_number, email, gender, raw_json, hospital_id
FROM patients WHERE id = $1`, id)

	return scanPatientRow(row)
}

// GetByIdentifier finds a patient by national_id OR passport_id (input can be either).
// Returns (nil, nil) if not found.
func (r *PatientRepo) GetByIdentifier(ctx context.Context, identifier string) (*Patient, error) {
	row := r.pool.QueryRow(ctx, `
SELECT id, patient_hn, national_id, passport_id,
       first_name_th, middle_name_th, last_name_th,
       first_name_en, middle_name_en, last_name_en,
       date_of_birth, phone_number, email, gender, raw_json, hospital_id
FROM patients WHERE national_id = $1 OR passport_id = $1 LIMIT 1`, identifier)

	return scanPatientRow(row)
}

// Create inserts a new patient
func (r *PatientRepo) Create(ctx context.Context, p *Patient) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO patients (
			id, patient_hn, national_id, passport_id,
			first_name_th, middle_name_th, last_name_th,
			first_name_en, middle_name_en, last_name_en,
			date_of_birth, phone_number, email, gender, raw_json, hospital_id
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)`,
		p.ID, p.PatientHN, p.NationalID, p.PassportID,
		p.FirstNameTH, p.MiddleNameTH, p.LastNameTH,
		p.FirstNameEN, p.MiddleNameEN, p.LastNameEN,
		p.DateOfBirth, p.PhoneNumber, p.Email, p.Gender, p.RawJSON, p.HospitalID,
	)
	return err
}

// Upsert inserts a patient or updates an existing record matching national_id or passport_id.
// Strategy:
//   - If national_id present -> ON CONFLICT (national_id) DO UPDATE ...
//   - Else if passport_id present -> ON CONFLICT (passport_id) DO UPDATE ...
//   - Else fallback to Create()
func (r *PatientRepo) Upsert(ctx context.Context, p *Patient) error {
	cols := []string{
		"id", "patient_hn", "national_id", "passport_id",
		"first_name_th", "middle_name_th", "last_name_th",
		"first_name_en", "middle_name_en", "last_name_en",
		"date_of_birth", "phone_number", "email", "gender", "raw_json", "hospital_id",
	}
	args := []any{
		p.ID, p.PatientHN, p.NationalID, p.PassportID,
		p.FirstNameTH, p.MiddleNameTH, p.LastNameTH,
		p.FirstNameEN, p.MiddleNameEN, p.LastNameEN,
		p.DateOfBirth, p.PhoneNumber, p.Email, p.Gender, p.RawJSON, p.HospitalID,
	}

	colsStr := strings.Join(cols, ",")
	// placeholders $1,$2,... up to len(args)
	ph := make([]string, len(args))
	for i := range ph {
		ph[i] = fmt.Sprintf("$%d", i+1)
	}
	placeholders := strings.Join(ph, ",")

	if p.NationalID != "" {
		finalSQL := fmt.Sprintf(
			"INSERT INTO patients (%s) VALUES (%s) "+
				"ON CONFLICT (national_id) DO UPDATE SET "+
				"patient_hn = EXCLUDED.patient_hn, "+
				"passport_id = COALESCE(EXCLUDED.passport_id, patients.passport_id), "+
				"first_name_th = EXCLUDED.first_name_th, middle_name_th = EXCLUDED.middle_name_th, last_name_th = EXCLUDED.last_name_th, "+
				"first_name_en = EXCLUDED.first_name_en, middle_name_en = EXCLUDED.middle_name_en, last_name_en = EXCLUDED.last_name_en, "+
				"date_of_birth = EXCLUDED.date_of_birth, phone_number = EXCLUDED.phone_number, email = EXCLUDED.email, gender = EXCLUDED.gender, "+
				"raw_json = EXCLUDED.raw_json, hospital_id = EXCLUDED.hospital_id",
			colsStr, placeholders,
		)
		_, err := r.pool.Exec(ctx, finalSQL, args...)
		return err
	} else if p.PassportID != "" {
		finalSQL := fmt.Sprintf(
			"INSERT INTO patients (%s) VALUES (%s) "+
				"ON CONFLICT (passport_id) DO UPDATE SET "+
				"patient_hn = EXCLUDED.patient_hn, "+
				"national_id = COALESCE(EXCLUDED.national_id, patients.national_id), "+
				"first_name_th = EXCLUDED.first_name_th, middle_name_th = EXCLUDED.middle_name_th, last_name_th = EXCLUDED.last_name_th, "+
				"first_name_en = EXCLUDED.first_name_en, middle_name_en = EXCLUDED.middle_name_en, last_name_en = EXCLUDED.last_name_en, "+
				"date_of_birth = EXCLUDED.date_of_birth, phone_number = EXCLUDED.phone_number, email = EXCLUDED.email, gender = EXCLUDED.gender, "+
				"raw_json = EXCLUDED.raw_json, hospital_id = EXCLUDED.hospital_id",
			colsStr, placeholders,
		)
		_, err := r.pool.Exec(ctx, finalSQL, args...)
		return err
	}

	return r.Create(ctx, p)
}

// scanPatientRow scans pgx.Row into Patient (used by GetByIdentifier and others).
func scanPatientRow(row pgx.Row) (*Patient, error) {
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
		&p.HospitalID,
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

// Filters for searching patients.
type PatientFilters struct {
	NationalID  string
	PassportID  string
	FirstName   string
	MiddleName  string
	LastName    string
	DateOfBirth string
	PhoneNumber string
	Email       string
}

// SearchPatients searches patients by optional filters and restricts by hospital_id.
// Returns (results, totalCount, error).
func (r *PatientRepo) SearchPatients(ctx context.Context, hospitalID string, f PatientFilters, limit, offset int) ([]*Patient, int, error) {
	// Build WHERE clauses with predictable parameter ordering:
	where := []string{"hospital_id = $1"}
	args := []any{hospitalID}
	idx := 2

	addEq := func(col string, val string) {
		where = append(where, fmt.Sprintf("%s = $%d", col, idx))
		args = append(args, val)
		idx++
	}

	addLike := func(col string, val string) {
		where = append(where, fmt.Sprintf("%s ILIKE $%d", col, idx))
		args = append(args, "%"+val+"%")
		idx++
	}

	if f.NationalID != "" {
		addEq("national_id", f.NationalID)
	}
	if f.PassportID != "" {
		addEq("passport_id", f.PassportID)
	}
	if f.FirstName != "" {
		addLike("first_name_en", f.FirstName)
		addLike("first_name_th", f.FirstName)
	}
	if f.MiddleName != "" {
		addLike("middle_name_en", f.MiddleName)
		addLike("middle_name_th", f.MiddleName)
	}
	if f.LastName != "" {
		addLike("last_name_en", f.LastName)
		addLike("last_name_th", f.LastName)
	}
	if f.DateOfBirth != "" {
		addEq("date_of_birth", f.DateOfBirth)
	}
	if f.PhoneNumber != "" {
		addLike("phone_number", f.PhoneNumber)
	}
	if f.Email != "" {
		addLike("email", f.Email)
	}

	whereClause := strings.Join(where, " AND ")

	// Count query
	countQuery := "SELECT COUNT(1) FROM patients WHERE " + whereClause
	var total int
	row := r.pool.QueryRow(ctx, countQuery, args...)
	if err := row.Scan(&total); err != nil {
		return nil, 0, err
	}

	// Prepare args for SELECT: original args followed by limit and offset
	argsForSelect := make([]any, 0, len(args)+2)
	argsForSelect = append(argsForSelect, args...)
	argsForSelect = append(argsForSelect, limit, offset)

	// placeholders for limit/offset are the next positions after original args
	limitPos := len(args) + 1  // e.g. if args had 2 items, limit is $3
	offsetPos := len(args) + 2 // offset is $4

	selectQuery := fmt.Sprintf(
		"SELECT id, patient_hn, national_id, passport_id, first_name_th, middle_name_th, last_name_th, first_name_en, middle_name_en, last_name_en, date_of_birth, phone_number, email, gender, raw_json FROM patients WHERE %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d",
		whereClause, limitPos, offsetPos,
	)

	rows, err := r.pool.Query(ctx, selectQuery, argsForSelect...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var results []*Patient
	for rows.Next() {
		var p Patient
		var dob sql.NullTime
		var raw []byte
		if err := rows.Scan(
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
		); err != nil {
			return nil, 0, err
		}
		if dob.Valid {
			str := dob.Time.Format("2006-01-02")
			p.DateOfBirth = &str
		}
		p.RawJSON = raw
		results = append(results, &p)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return results, total, nil
}
