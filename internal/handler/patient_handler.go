package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/haniscreator/agnos-search/internal/repository"
)

// PatientService defines the minimal service used by the handlers.
type PatientService interface {
	Get(ctx context.Context, identifier string) (*repository.Patient, error)
	Search(ctx context.Context, hospitalID string, filters repository.PatientFilters, limit, offset int) ([]*repository.Patient, int, error)
}

// RegisterPatientRoutes registers patient-related routes on the provided Gin router.
// Note: analytics can be nil if audit logging is not desired.
func RegisterPatientRoutes(r gin.IRoutes, svc PatientService, analytics repository.AnalyticsRepo) {
	// GET /v1/patient/:id
	r.GET("/v1/patient/:id", func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "id is required"})
			return
		}

		p, err := svc.Get(c.Request.Context(), id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
			return
		}
		if p == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}

		hidVal, exists := c.Get("hospital_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing hospital in token"})
			return
		}
		hid, ok := hidVal.(string)
		if !ok || hid == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid hospital in token"})
			return
		}

		if p.HospitalID != "" && p.HospitalID != hid {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}

		c.JSON(http.StatusOK, p)
	})

	// POST /patient/search
	r.POST("/patient/search", func(c *gin.Context) {
		hv, ok := c.Get("hospital_id")
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing hospital in token"})
			return
		}
		hid, ok := hv.(string)
		if !ok || hid == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid hospital in token"})
			return
		}

		var req struct {
			NationalID  string `json:"national_id"`
			PassportID  string `json:"passport_id"`
			FirstName   string `json:"first_name"`
			MiddleName  string `json:"middle_name"`
			LastName    string `json:"last_name"`
			DateOfBirth string `json:"date_of_birth"`
			PhoneNumber string `json:"phone_number"`
			Email       string `json:"email"`
			Limit       int    `json:"limit"`
			Offset      int    `json:"offset"`
		}
		if err := c.BindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
			return
		}
		if req.Limit == 0 {
			req.Limit = 10
		}

		f := repository.PatientFilters{
			NationalID:  req.NationalID,
			PassportID:  req.PassportID,
			FirstName:   req.FirstName,
			MiddleName:  req.MiddleName,
			LastName:    req.LastName,
			DateOfBirth: req.DateOfBirth,
			PhoneNumber: req.PhoneNumber,
			Email:       req.Email,
		}

		results, total, err := svc.Search(c.Request.Context(), hid, f, req.Limit, req.Offset)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal"})
			return
		}

		// Log audit event if analytics repo provided.
		if analytics != nil {
			// try best-effort: don't block the response if audit fails.
			if staffVal, ok := c.Get("staff_id"); ok {
				if staffID, _ok := staffVal.(string); _ok {
					// logging in a goroutine so it doesn't delay response
					go func() {
						_ = analytics.LogSearch(context.Background(), staffID, hid, f, total)
					}()
				}
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"count":   total,
			"limit":   req.Limit,
			"offset":  req.Offset,
			"results": results,
		})
	})
}
