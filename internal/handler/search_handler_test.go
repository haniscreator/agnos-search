package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/haniscreator/agnos-search/internal/repository"
)

// mockService implements PatientService for tests
type mockPatientService struct {
	out   []*repository.Patient
	total int
	err   error
}

func (m *mockPatientService) Get(ctx context.Context, identifier string) (*repository.Patient, error) {
	return nil, nil
}

func (m *mockPatientService) Search(ctx context.Context, hospitalID string, filters repository.PatientFilters, limit, offset int) ([]*repository.Patient, int, error) {
	return m.out, m.total, m.err
}

func TestSearchHandler_ReturnsResults(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	mock := &mockPatientService{
		out: []*repository.Patient{
			{
				ID:          "p1",
				PatientHN:   "HN-1",
				NationalID:  "N-1",
				FirstNameEN: "Somchai",
			},
		},
		total: 1,
		err:   nil,
	}

	// IMPORTANT: insert test middleware to simulate JWT middleware setting hospital_id
	r.Use(func(c *gin.Context) {
		c.Set("hospital_id", "HIS-1")
		c.Next()
	})

	// Register routes AFTER middleware so handler sees hospital_id
	RegisterPatientRoutes(r, mock)

	// Build request
	body := `{"national_id":"N-1","limit":10,"offset":0}`
	req := httptest.NewRequest(http.MethodPost, "/patient/search", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Somchai")
}
