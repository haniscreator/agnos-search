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
}

// RegisterPatientRoutes attaches patient routes to the provided Gin engine.
func RegisterPatientRoutes(r *gin.Engine, svc PatientService) {
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

		// Map repository Patient to API response
		resp := gin.H{
			"first_name_th":  p.FirstNameTH,
			"middle_name_th": p.MiddleNameTH,
			"last_name_th":   p.LastNameTH,
			"first_name_en":  p.FirstNameEN,
			"middle_name_en": p.MiddleNameEN,
			"last_name_en":   p.LastNameEN,
			"date_of_birth":  p.DateOfBirth,
			"patient_hn":     p.PatientHN,
			"national_id":    p.NationalID,
			"passport_id":    p.PassportID,
			"phone_number":   p.PhoneNumber,
			"email":          p.Email,
			"gender":         p.Gender,
		}
		c.JSON(http.StatusOK, resp)
	})
}
