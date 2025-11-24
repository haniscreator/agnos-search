package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/haniscreator/agnos-search/internal/adapter"
	"github.com/haniscreator/agnos-search/internal/db"
	"github.com/haniscreator/agnos-search/internal/handler"
	"github.com/haniscreator/agnos-search/internal/middleware"
	"github.com/haniscreator/agnos-search/internal/repository"
	"github.com/haniscreator/agnos-search/internal/service"
	"github.com/joho/godotenv"
)

func main() {

	// -------------------------------
	// Load .env (ignore errors if file is missing)
	// -------------------------------
	godotenv.Load()
	if err := godotenv.Load(); err == nil {
		if os.Getenv("JWT_SECRET") != "" {
			log.Println("Loaded environment .env (JWT_SECRET detected)")
		} else {
			log.Println("Loaded environment .env")
		}
	}

	r := gin.Default()

	// Basic endpoints
	r.GET("/health", healthHandler)
	r.GET("/v1/search", searchHandler)

	// Create DB pool
	ctx := context.Background()
	pool, err := db.NewPool(ctx)
	if err != nil {
		log.Printf("warning: could not create db pool: %v; some routes will return 500", err)
		registerStubs(r)
	} else {
		// repositories
		staffRepo := repository.NewStaffRepo(pool)
		patientRepo := repository.NewPatientRepo(pool)
		analyticsRepo := repository.NewAnalyticsRepo(pool)

		// auth service
		authSvc := service.NewAuthService(staffRepo)
		handler.RegisterAuthRoutes(r, authSvc)

		// adapter + patient service
		base := os.Getenv("HOSPITAL_BASE")
		if base == "" {
			base = "http://hospital-a.api.co.th"
		}
		adapterClient, aErr := adapter.NewHospitalAdapter(base, 2*time.Second)
		if aErr != nil {
			log.Printf("warning: could not create adapter: %v; patient routes will use stub", aErr)
			stub := &dbUnavailableService{err: fmt.Errorf("hospital adapter not available")}
			handler.RegisterPatientRoutes(r, stub, analyticsRepo)
		} else {
			patientSvc := service.NewPatientService(patientRepo, adapterClient)
			jwtSecret := os.Getenv("JWT_SECRET")
			if jwtSecret == "" {
				log.Fatal("FATAL: JWT_SECRET is missing. Set it in .env or docker-compose!")
			}
			authGroup := r.Group("/")
			authGroup.Use(middleware.AuthMiddleware(jwtSecret))

			handler.RegisterPatientRoutes(authGroup, patientSvc, analyticsRepo)
		}
	}

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("failed to run server: %v", err)
	}
}

func healthHandler(c *gin.Context) {
	c.JSON(200, gin.H{"status": "ok"})
}

func searchHandler(c *gin.Context) {
	q := c.Query("q")
	if q == "" {
		c.JSON(400, gin.H{"error": "q is required"})
		return
	}
	results := []gin.H{{"type": "patient", "id": "p_1", "name": "Demo Patient"}}
	c.JSON(200, gin.H{"query": q, "results": results})
}

func registerStubs(r *gin.Engine) {
	r.POST("/staff/create", func(c *gin.Context) {
		c.JSON(500, gin.H{"error": "database not available"})
	})
	r.POST("/staff/login", func(c *gin.Context) {
		c.JSON(500, gin.H{"error": "database not available"})
	})
}

type dbUnavailableService struct {
	err error
}

func (s *dbUnavailableService) Get(_ context.Context, _ string) (*repository.Patient, error) {
	return nil, s.err
}

func (s *dbUnavailableService) Search(_ context.Context, _ string, _ repository.PatientFilters, _ int, _ int) ([]*repository.Patient, int, error) {
	return nil, 0, s.err
}
