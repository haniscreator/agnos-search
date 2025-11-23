package adapter

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"time"

	"github.com/haniscreator/agnos-search/internal/repository"
)

// HospitalClient is the interface the rest of the app will depend on.
type HospitalClient interface {
	// LookupByIdentifier queries the hospital API by national_id or passport_id.
	// Returns (nil, nil) if hospital returns 404 (not found).
	LookupByIdentifier(ctx context.Context, identifier string) (*repository.Patient, error)
}

// HospitalAdapter calls a hospital HTTP API and maps the response to repository.Patient.
type HospitalAdapter struct {
	baseURL    *url.URL
	httpClient *http.Client
}

// NewHospitalAdapter constructs a HospitalAdapter.
// base := "https://hospital-a.api.co.th" (no trailing slash required).
// timeout controls the HTTP client request timeout.
func NewHospitalAdapter(base string, timeout time.Duration) (*HospitalAdapter, error) {
	u, err := url.Parse(base)
	if err != nil {
		return nil, fmt.Errorf("invalid base url: %w", err)
	}
	c := &http.Client{
		Timeout: timeout,
	}
	return &HospitalAdapter{
		baseURL:    u,
		httpClient: c,
	}, nil
}

// expected hospital response JSON structure (partial mapping)
type hospitalResponse struct {
	FirstNameTH  string `json:"first_name_th"`
	MiddleNameTH string `json:"middle_name_th"`
	LastNameTH   string `json:"last_name_th"`

	FirstNameEN  string `json:"first_name_en"`
	MiddleNameEN string `json:"middle_name_en"`
	LastNameEN   string `json:"last_name_en"`

	DateOfBirth string `json:"date_of_birth"`
	PatientHN   string `json:"patient_hn"`

	NationalID string `json:"national_id"`
	PassportID string `json:"passport_id"`

	PhoneNumber string `json:"phone_number"`
	Email       string `json:"email"`
	Gender      string `json:"gender"`
}

// LookupByIdentifier implements HospitalClient.
func (h *HospitalAdapter) LookupByIdentifier(ctx context.Context, identifier string) (*repository.Patient, error) {
	// build URL: base + /patient/search/{id}
	u := *h.baseURL // copy
	u.Path = path.Join(h.baseURL.Path, "patient", "search", identifier)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	// 404 -> not found
	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return nil, fmt.Errorf("hospital api status %d: %s", resp.StatusCode, string(body))
	}

	// read body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	var hr hospitalResponse
	if err := json.Unmarshal(body, &hr); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	// Map to repository.Patient
	p := &repository.Patient{
		// ID left empty here; repository.Create should set a UUID if needed
		PatientHN:    hr.PatientHN,
		NationalID:   hr.NationalID,
		PassportID:   hr.PassportID,
		FirstNameTH:  hr.FirstNameTH,
		MiddleNameTH: hr.MiddleNameTH,
		LastNameTH:   hr.LastNameTH,
		FirstNameEN:  hr.FirstNameEN,
		MiddleNameEN: hr.MiddleNameEN,
		LastNameEN:   hr.LastNameEN,
		DateOfBirth:  nil,
		PhoneNumber:  hr.PhoneNumber,
		Email:        hr.Email,
		Gender:       hr.Gender,
		RawJSON:      body,
	}
	if hr.DateOfBirth != "" {
		p.DateOfBirth = &hr.DateOfBirth
	}

	return p, nil
}
