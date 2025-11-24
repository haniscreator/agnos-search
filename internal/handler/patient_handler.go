package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/haniscreator/agnos-search/internal/repository"
)

// PatientService is the minimal service interface the handler needs.
type PatientService interface {
	Get(ctx context.Context, identifier string) (*repository.Patient, error)
	Search(ctx context.Context, hospitalID string, filters repository.PatientFilters, limit, offset int) ([]*repository.Patient, int, error)
}

// RegisterPatientRoutes attaches patient routes to the provided router (Engine or RouterGroup).
func RegisterPatientRoutes(r gin.IRouter, svc PatientService) {
	r.GET("/v1/patient/:id", func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "id is required"})
			return
		}

		p, err := svc.Get(c.Request.Context(), id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal"})
			return
		}
		if p == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "patient not found"})
			return
		}

		resp := patientToResp(p)
		c.JSON(http.StatusOK, resp)
	})

	// Protected search route (expects middleware to have set hospital_id in context)
	r.POST("/patient/search", func(c *gin.Context) {
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
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request", "detail": err.Error()})
			return
		}

		// default pagination
		limit := req.Limit
		if limit <= 0 || limit > 100 {
			limit = 20
		}
		offset := req.Offset
		if offset < 0 {
			offset = 0
		}

		// get hospital_id from context (set by JWT middleware)
		hidVal, ok := c.Get("hospital_id")
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing hospital in token"})
			return
		}
		hospitalID, _ := hidVal.(string)

		filters := repository.PatientFilters{
			NationalID:  req.NationalID,
			PassportID:  req.PassportID,
			FirstName:   req.FirstName,
			MiddleName:  req.MiddleName,
			LastName:    req.LastName,
			DateOfBirth: req.DateOfBirth,
			PhoneNumber: req.PhoneNumber,
			Email:       req.Email,
		}

		results, total, err := svc.Search(c.Request.Context(), hospitalID, filters, limit, offset)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal"})
			return
		}

		// map results
		out := make([]gin.H, 0, len(results))
		for _, p := range results {
			out = append(out, patientToResp(p))
		}

		c.JSON(http.StatusOK, gin.H{
			"count":   total,
			"limit":   limit,
			"offset":  offset,
			"results": out,
		})
	})
}

func patientToResp(p *repository.Patient) gin.H {
	return gin.H{
		"id":             p.ID,
		"patient_hn":     p.PatientHN,
		"first_name_th":  p.FirstNameTH,
		"middle_name_th": p.MiddleNameTH,
		"last_name_th":   p.LastNameTH,
		"first_name_en":  p.FirstNameEN,
		"middle_name_en": p.MiddleNameEN,
		"last_name_en":   p.LastNameEN,
		"date_of_birth":  p.DateOfBirth,
		"national_id":    p.NationalID,
		"passport_id":    p.PassportID,
		"phone_number":   p.PhoneNumber,
		"email":          p.Email,
		"gender":         p.Gender,
	}
}
