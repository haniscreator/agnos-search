package handler

import (
	"context"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/haniscreator/agnos-search/internal/repository"
)

// PatientService defines the minimal service used by the handlers.
type PatientService interface {
	Get(ctx context.Context, identifier string) (*repository.Patient, error)
	Search(ctx context.Context, hospitalID string, filters repository.PatientFilters, limit, offset int) ([]*repository.Patient, int, error)
}

// PatientWriter defines minimal write operations for patients (used by create endpoint).
type PatientWriter interface {
	Upsert(ctx context.Context, p *repository.Patient) error
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
			log.Printf("patient/get error (id=%s): %v", id, err)
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
			log.Printf("patient/search bind error: %v", err)
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
			log.Printf(
				"patient/search service error (hospital=%s, filters=%+v, limit=%d, offset=%d): %v",
				hid, f, req.Limit, req.Offset, err,
			)
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
						if err := analytics.LogSearch(context.Background(), staffID, hid, f, total); err != nil {
							log.Printf(
								"patient/search analytics error (staff_id=%s, hospital=%s): %v",
								staffID, hid, err,
							)
						}
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

// RegisterPatientWriteRoutes registers write endpoints like POST /v1/patients.
func RegisterPatientWriteRoutes(r gin.IRoutes, writer PatientWriter) {
	// POST /v1/patients - create (or upsert) a patient
	r.POST("/v1/patients", func(c *gin.Context) {
		// Get hospital from JWT (already set by AuthMiddleware)
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

		// Request payload for creating a patient
		var req struct {
			PatientHN    string `json:"patient_hn"`
			NationalID   string `json:"national_id"`
			PassportID   string `json:"passport_id"`
			FirstNameTH  string `json:"first_name_th"`
			MiddleNameTH string `json:"middle_name_th"`
			LastNameTH   string `json:"last_name_th"`
			FirstNameEN  string `json:"first_name_en"`
			MiddleNameEN string `json:"middle_name_en"`
			LastNameEN   string `json:"last_name_en"`
			DateOfBirth  string `json:"date_of_birth"` // yyyy-mm-dd
			PhoneNumber  string `json:"phone_number"`
			Email        string `json:"email"`
			Gender       string `json:"gender"` // "M", "F", etc.
		}

		if err := c.BindJSON(&req); err != nil {
			log.Printf("patient/create bind error: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
			return
		}

		// Optional: basic validation – at least one identifier
		if req.NationalID == "" && req.PassportID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "national_id or passport_id is required"})
			return
		}

		// Map to repository.Patient
		var dob *string
		if req.DateOfBirth != "" {
			d := req.DateOfBirth
			dob = &d
		}

		p := &repository.Patient{
			ID:           uuid.NewString(),
			PatientHN:    req.PatientHN,
			NationalID:   req.NationalID,
			PassportID:   req.PassportID,
			FirstNameTH:  req.FirstNameTH,
			MiddleNameTH: req.MiddleNameTH,
			LastNameTH:   req.LastNameTH,
			FirstNameEN:  req.FirstNameEN,
			MiddleNameEN: req.MiddleNameEN,
			LastNameEN:   req.LastNameEN,
			DateOfBirth:  dob,
			PhoneNumber:  req.PhoneNumber,
			Email:        req.Email,
			Gender:       req.Gender,
			RawJSON:      nil,
			HospitalID:   hid,
		}

		if err := writer.Upsert(c.Request.Context(), p); err != nil {
			log.Printf("patient/create Upsert ERROR DETAILS: %+v", err)

			// Return error message in response temporarily (for debugging)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "upsert_failed",
				"details": err.Error(),
			})
			return
		}

		c.JSON(http.StatusCreated, p)
	})
}
