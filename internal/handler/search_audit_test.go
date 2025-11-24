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

type mockAnalytics struct {
	called     bool
	lastCount  int
	lastStaff  string
	lastHosp   string
	lastFilter repository.PatientFilters
}

func (m *mockAnalytics) LogSearch(ctx context.Context, staffID, hospitalID string, filters repository.PatientFilters, resultCount int) error {
	m.called = true
	m.lastCount = resultCount
	m.lastStaff = staffID
	m.lastHosp = hospitalID
	m.lastFilter = filters
	return nil
}

func TestSearchHandler_AuditLogged(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	// mock patient service returning 1 result
	mockSvc := &mockService{
		sout: []*repository.Patient{
			{ID: "p1", PatientHN: "HN-1", NationalID: "N-1", FirstNameEN: "Somchai"},
		},
		total: 1,
	}

	// create analytics mock
	ma := &mockAnalytics{}

	// stub middleware to inject staff_id and hospital_id
	r.Use(func(c *gin.Context) {
		c.Set("hospital_id", "HIS-1")
		c.Set("staff_id", "staff-1")
		c.Next()
	})

	// register routes with analytics mock
	RegisterPatientRoutes(r, mockSvc, ma)

	body := `{"national_id":"N-1","limit":10,"offset":0}`
	req := httptest.NewRequest(http.MethodPost, "/patient/search", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	// audit should have been invoked (goroutine may have raced, but in unit test it will have run quickly)
	// to be conservative, check the mock flags directly (it sets them synchronously)
	assert.True(t, ma.called, "analytics LogSearch should be called")
	assert.Equal(t, 1, ma.lastCount)
	assert.Equal(t, "staff-1", ma.lastStaff)
	assert.Equal(t, "HIS-1", ma.lastHosp)
}
