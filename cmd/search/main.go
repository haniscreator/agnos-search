package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	"github.com/haniscreator/agnos-search/internal/adapter"
	"github.com/haniscreator/agnos-search/internal/db"
	"github.com/haniscreator/agnos-search/internal/handler"
	"github.com/haniscreator/agnos-search/internal/middleware"
	"github.com/haniscreator/agnos-search/internal/repository"
	"github.com/haniscreator/agnos-search/internal/service"
)

func main() {
	// Load .env if present (useful both locally and in Docker when bind-mounted)
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

	// Create DB pool with retries (important for CI / fresh docker-compose)
	ctx := context.Background()
	var (
		pool *pgxpool.Pool
		err  error
	)

	const maxAttempts = 5
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		pool, err = db.NewPool(ctx)
		if err == nil {
			log.Printf("database pool created (attempt %d)", attempt)
			break
		}
		log.Printf("database not ready (attempt %d/%d): %v", attempt, maxAttempts, err)
		time.Sleep(2 * time.Second)
	}

	if err != nil {
		log.Printf("warning: could not create db pool after retries: %v; some routes will return 500", err)
		// register routes with stubs that return 500 for DB-dependent endpoints
		registerStubs(r)
	} else {
		// repositories
		staffRepo := repository.NewStaffRepo(pool)
		patientRepo := repository.NewPatientRepo(pool)
		analyticsRepo := repository.NewAnalyticsRepo(pool)

		// auth service
		authSvc := service.NewAuthService(staffRepo)
		// register auth routes
		handler.RegisterAuthRoutes(r, authSvc)

		// adapter + patient service (reuse existing wiring)
		base := os.Getenv("HOSPITAL_BASE")
		if base == "" {
			base = "http://hospital-a.api.co.th"
		}
		adapterClient, aErr := adapter.NewHospitalAdapter(base, 2*time.Second)
		if aErr != nil {
			log.Printf("warning: could not create adapter: %v; patient routes will use stub", aErr)
			// still register patient route with stub service
			stub := &dbUnavailableService{err: fmt.Errorf("hospital adapter not available")}
			handler.RegisterPatientRoutes(r, stub, analyticsRepo)
		} else {
			patientSvc := service.NewPatientService(patientRepo, adapterClient)
			jwtSecret := os.Getenv("JWT_SECRET")
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
	// simple stub endpoints that respond 500 for DB dependent routes
	r.POST("/staff/create", func(c *gin.Context) {
		c.JSON(500, gin.H{"error": "database not available"})
	})
	r.POST("/staff/login", func(c *gin.Context) {
		c.JSON(500, gin.H{"error": "database not available"})
	})
	// keep patient route registered earlier in other code
}

// dbUnavailableService returns errors when DB or adapter not available.
type dbUnavailableService struct {
	err error
}

func (s *dbUnavailableService) Get(_ context.Context, _ string) (*repository.Patient, error) {
	return nil, s.err
}

func (s *dbUnavailableService) Search(_ context.Context, _ string, _ repository.PatientFilters, _ int, _ int) ([]*repository.Patient, int, error) {
	return nil, 0, s.err
}
