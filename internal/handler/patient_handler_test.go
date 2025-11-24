package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/haniscreator/agnos-search/internal/repository"
)

// mockService satisfies PatientService (Get + Search)
type mockService struct {
	out   *repository.Patient
	sout  []*repository.Patient
	total int
	err   error
}

func (m *mockService) Get(_ context.Context, identifier string) (*repository.Patient, error) {
	return m.out, m.err
}

func (m *mockService) Search(_ context.Context, hospitalID string, filters repository.PatientFilters, limit, offset int) ([]*repository.Patient, int, error) {
	return m.sout, m.total, m.err
}

// setupRouterWithMock returns a new Gin engine for tests.
// It does NOT register routes so tests can set middleware before registration.
func setupRouterWithMock(m PatientService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	return r
}

func TestGetPatient_Found(t *testing.T) {
	mock := &mockService{
		out: &repository.Patient{
			ID:           "uuid-1",
			PatientHN:    "HN-123",
			NationalID:   "N-123",
			PassportID:   "P-123",
			FirstNameTH:  "สมชาย",
			MiddleNameTH: "",
			LastNameTH:   "ใจดี",
			FirstNameEN:  "Somchai",
			LastNameEN:   "Jaidee",
			DateOfBirth:  strptr("1990-01-01"),
			PhoneNumber:  "0812345678",
			Email:        "a@example.com",
			Gender:       "M",
			HospitalID:   "HIS-1",
		},
		err: nil,
	}
	r := setupRouterWithMock(mock)

	// set middleware that simulates the JWT middleware (hospital_id present)
	r.Use(func(c *gin.Context) {
		c.Set("hospital_id", "HIS-1")
		c.Next()
	})

	// Register routes AFTER middleware so hospital_id is visible to handlers.
	RegisterPatientRoutes(r, mock, nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/patient/N-123", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Somchai")
	assert.Contains(t, w.Body.String(), "HN-123")
}

func TestGetPatient_NotFound(t *testing.T) {
	mock := &mockService{out: nil, err: nil}
	r := setupRouterWithMock(mock)

	// middleware present but not strictly necessary for not found case
	r.Use(func(c *gin.Context) {
		c.Set("hospital_id", "HIS-1")
		c.Next()
	})

	RegisterPatientRoutes(r, mock, nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/patient/missing", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetPatient_Error(t *testing.T) {
	mock := &mockService{out: nil, err: errExample()}
	r := setupRouterWithMock(mock)

	r.Use(func(c *gin.Context) {
		c.Set("hospital_id", "HIS-1")
		c.Next()
	})

	RegisterPatientRoutes(r, mock, nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/patient/any", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// helpers
func strptr(s string) *string { return &s }
func errExample() error       { return &customErr{"boom"} }

type customErr struct{ s string }

func (e *customErr) Error() string { return e.s }
