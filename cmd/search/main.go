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
	// 1) Load .env early
	if err := godotenv.Load(); err == nil {
		if os.Getenv("JWT_SECRET") != "" {
			log.Println("Loaded environment .env (JWT_SECRET detected)")
		} else {
			log.Println("Loaded environment .env")
		}
	} else {
		log.Println("No .env file found or could not load .env (this may be fine if env is injected)")
	}

	// 2) Create DB pool with retries (BLOCK until success or fatal)
	ctx := context.Background()
	pool := mustCreateDBPoolWithRetry(ctx)

	// 3) Wire repositories & services
	staffRepo := repository.NewStaffRepo(pool)
	patientRepo := repository.NewPatientRepo(pool)
	analyticsRepo := repository.NewAnalyticsRepo(pool)

	authSvc := service.NewAuthService(staffRepo)

	base := os.Getenv("HOSPITAL_BASE")
	if base == "" {
		base = "http://hospital-a.api.co.th"
	}

	adapterClient, aErr := adapter.NewHospitalAdapter(base, 2*time.Second)

	// 4) Setup Gin AFTER all deps are ready
	r := gin.Default()

	// basic endpoints that don't need DB
	r.GET("/health", healthHandler)
	r.GET("/v1/search", searchHandler)

	// auth routes (staff/create, staff/login) - DB-backed
	handler.RegisterAuthRoutes(r, authSvc)

	// patient routes (behind JWT)
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Println("WARNING: JWT_SECRET is empty; auth middleware will reject all protected endpoints")
	}
	authGroup := r.Group("/")
	authGroup.Use(middleware.AuthMiddleware(jwtSecret))

	if aErr != nil {
		log.Printf("warning: could not create adapter: %v; patient routes will use stub", aErr)
		stub := &dbUnavailableService{err: fmt.Errorf("hospital adapter not available")}
		handler.RegisterPatientRoutes(authGroup, stub, analyticsRepo)
	} else {
		patientSvc := service.NewPatientService(patientRepo, adapterClient)
		handler.RegisterPatientRoutes(authGroup, patientSvc, analyticsRepo)
	}

	// 5) Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("failed to run server: %v", err)
	}
}

// mustCreateDBPoolWithRetry blocks until DB is ready or exits fatally.
func mustCreateDBPoolWithRetry(ctx context.Context) *pgxpool.Pool {
	const maxAttempts = 5

	var (
		pool *pgxpool.Pool
		err  error
	)

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		pool, err = db.NewPool(ctx)
		if err == nil {
			log.Printf("database pool created (attempt %d)", attempt)
			return pool
		}
		log.Printf("database not ready (attempt %d/%d): %v", attempt, maxAttempts, err)
		time.Sleep(2 * time.Second)
	}

	log.Fatalf("could not create db pool after %d attempts: %v", maxAttempts, err)
	return nil // unreachable
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
