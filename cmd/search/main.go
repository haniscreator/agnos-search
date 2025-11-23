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
	"github.com/haniscreator/agnos-search/internal/repository"
	"github.com/haniscreator/agnos-search/internal/service"
)

func main() {
	r := gin.Default()

	// Always register basic endpoints
	r.GET("/health", healthHandler)
	r.GET("/v1/search", searchHandler)

	// Try to create DB pool and register DB-backed routes + adapter-backed service
	ctx := context.Background()
	pool, err := db.NewPool(ctx)
	if err != nil {
		log.Printf("warning: could not create db pool: %v; patient routes will return 500 (DB unavailable)", err)
		stub := &dbUnavailableService{err: fmt.Errorf("database not available")}
		handler.RegisterPatientRoutes(r, stub)
	} else {
		repo := repository.NewPatientRepo(pool)

		// create adapter (base URL from env or default)
		base := os.Getenv("HOSPITAL_BASE")
		if base == "" {
			base = "http://hospital-a.api.co.th"
		}
		// short timeout; you can tune via env
		adapterClient, aErr := adapter.NewHospitalAdapter(base, 2*time.Second)
		if aErr != nil {
			// If adapter creation fails, still register stub service
			log.Printf("warning: could not create adapter: %v; patient routes will return 500", aErr)
			stub := &dbUnavailableService{err: fmt.Errorf("hospital adapter not available")}
			handler.RegisterPatientRoutes(r, stub)
		} else {
			svc := service.NewPatientService(repo, adapterClient)
			handler.RegisterPatientRoutes(r, svc)
		}
	}

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

// dbUnavailableService returns errors when DB or adapter not available.
type dbUnavailableService struct {
	err error
}

func (s *dbUnavailableService) Get(_ context.Context, _ string) (*repository.Patient, error) {
	return nil, s.err
}
