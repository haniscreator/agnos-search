package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/gin-gonic/gin"

	"github.com/haniscreator/agnos-search/internal/db"
	"github.com/haniscreator/agnos-search/internal/handler"
	"github.com/haniscreator/agnos-search/internal/repository"
)

func main() {
	r := gin.Default()

	// Always register basic endpoints
	r.GET("/health", healthHandler)
	r.GET("/v1/search", searchHandler)

	// try to create DB pool and register DB-backed routes
	ctx := context.Background()
	pool, err := db.NewPool(ctx)
	if err != nil {
		// If DB is not available in local dev, log and register a stub reader
		log.Printf("warning: could not create db pool: %v; patient routes will return 500", err)
		stub := &dbUnavailableReader{err: fmt.Errorf("database not available")}
		handler.RegisterPatientRoutes(r, stub)
	} else {
		// Use real repo when DB is available
		repo := repository.NewPatientRepo(pool)
		handler.RegisterPatientRoutes(r, repo)
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

	results := []gin.H{
		{"type": "patient", "id": "p_1", "name": "Demo Patient"},
	}
	c.JSON(200, gin.H{"query": q, "results": results})
}

// dbUnavailableReader is a small stub that implements the handler.PatientReader
// interface so the route exists even if the DB connection failed.
type dbUnavailableReader struct {
	err error
}

// GetByIdentifier returns an internal error indicating DB is not available.
func (s *dbUnavailableReader) GetByIdentifier(_ context.Context, _ string) (*repository.Patient, error) {
	return nil, s.err
}
