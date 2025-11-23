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

// mockReader satisfies PatientReader
type mockReader struct {
	out *repository.Patient
	err error
}

func (m *mockReader) GetByIdentifier(_ context.Context, identifier string) (*repository.Patient, error) {
	// simple behavior: return configured out
	return m.out, m.err
}

func setupRouterWithMock(m PatientReader) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	RegisterPatientRoutes(r, m)
	return r
}

func TestGetPatient_Found(t *testing.T) {
	mock := &mockReader{
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
		},
		err: nil,
	}
	r := setupRouterWithMock(mock)

	req := httptest.NewRequest(http.MethodGet, "/v1/patient/N-123", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Somchai")
	assert.Contains(t, w.Body.String(), "HN-123")
}

func TestGetPatient_NotFound(t *testing.T) {
	mock := &mockReader{out: nil, err: nil}
	r := setupRouterWithMock(mock)

	req := httptest.NewRequest(http.MethodGet, "/v1/patient/missing", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetPatient_Error(t *testing.T) {
	mock := &mockReader{out: nil, err: errExample()}
	r := setupRouterWithMock(mock)

	req := httptest.NewRequest(http.MethodGet, "/v1/patient/any", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// small helpers
func strptr(s string) *string { return &s }
func errExample() error       { return &customErr{"boom"} }

type customErr struct{ s string }

func (e *customErr) Error() string { return e.s }
