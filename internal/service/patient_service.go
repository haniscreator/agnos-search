package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/haniscreator/agnos-search/internal/adapter"
	"github.com/haniscreator/agnos-search/internal/repository"
)

// PatientService defines high-level behavior used by HTTP handlers.
type PatientService interface {
	Get(ctx context.Context, identifier string) (*repository.Patient, error)
	// Search with hospital constraint
	Search(ctx context.Context, hospitalID string, filters repository.PatientFilters, limit, offset int) ([]*repository.Patient, int, error)
}

// patientServiceImpl implements PatientService
type patientServiceImpl struct {
	repo    *repository.PatientRepo
	adapter adapter.HospitalClient
}

func NewPatientService(repo *repository.PatientRepo, adapter adapter.HospitalClient) PatientService {
	return &patientServiceImpl{repo: repo, adapter: adapter}
}

func (s *patientServiceImpl) Get(ctx context.Context, identifier string) (*repository.Patient, error) {
	// 1) Try DB
	p, err := s.repo.GetByIdentifier(ctx, identifier)
	if err != nil {
		return nil, fmt.Errorf("repo get: %w", err)
	}
	if p != nil {
		return p, nil
	}

	// 2) Query hospital adapter
	p, err = s.adapter.LookupByIdentifier(ctx, identifier)
	if err != nil {
		return nil, fmt.Errorf("adapter lookup: %w", err)
	}
	if p == nil {
		// not found at hospital either
		return nil, nil
	}

	// 3) Persist to DB (set HospitalID from context if available) -- prefer context value "hospital_id"
	if p.HospitalID == "" {
		if hid, ok := ctx.Value("hospital_id").(string); ok && hid != "" {
			p.HospitalID = hid
		}
	}
	if p.ID == "" {
		p.ID = uuid.NewString()
	}

	// use Upsert so adapter results update existing rows instead of inserting duplicates
	if err := s.repo.Upsert(ctx, p); err != nil {
		return nil, fmt.Errorf("repo upsert: %w", err)
	}

	return p, nil
}

func (s *patientServiceImpl) Search(ctx context.Context, hospitalID string, filters repository.PatientFilters, limit, offset int) ([]*repository.Patient, int, error) {
	results, total, err := s.repo.SearchPatients(ctx, hospitalID, filters, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("repo search: %w", err)
	}
	return results, total, nil
}
