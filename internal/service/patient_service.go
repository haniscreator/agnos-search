package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/haniscreator/agnos-search/internal/adapter"
	"github.com/haniscreator/agnos-search/internal/repository"
)

// PatientService defines the high-level behavior used by HTTP handlers.
type PatientService interface {
	// Get looks up a patient by identifier (national_id or passport_id).
	// It first checks DB; if not found, it queries HospitalAdapter and persists the result.
	Get(ctx context.Context, identifier string) (*repository.Patient, error)
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

	// 3) Persist to DB (set ID if missing)
	if p.ID == "" {
		p.ID = uuid.NewString()
	}
	if err := s.repo.Create(ctx, p); err != nil {
		return nil, fmt.Errorf("repo create: %w", err)
	}

	return p, nil
}
