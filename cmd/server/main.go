package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/malbinjvc/pixel-forge/internal/handler"
	"github.com/malbinjvc/pixel-forge/internal/middleware"
	"github.com/malbinjvc/pixel-forge/internal/storage"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	storageDir := os.Getenv("STORAGE_DIR")
	if storageDir == "" {
		storageDir = "./data/images"
	}

	store, err := storage.New(storageDir)
	if err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}

	h := handler.New(store)
	rateLimiter := middleware.NewRateLimiter(100, time.Minute)

	mux := http.NewServeMux()

	// Health
	mux.HandleFunc("GET /health", h.Health)

	// Upload
	mux.HandleFunc("POST /api/upload", h.Upload)

	// Image operations
	mux.HandleFunc("POST /api/images/{id}/resize", h.ProcessResize)
	mux.HandleFunc("POST /api/images/{id}/crop", h.ProcessCrop)
	mux.HandleFunc("POST /api/images/{id}/rotate", h.ProcessRotate)
	mux.HandleFunc("POST /api/images/{id}/filter", h.ProcessFilter)
	mux.HandleFunc("POST /api/images/{id}/convert", h.ProcessConvert)

	// Image retrieval
	mux.HandleFunc("GET /api/images/{id}", h.GetImage)

	// Jobs
	mux.HandleFunc("GET /api/jobs", h.ListJobs)
	mux.HandleFunc("GET /api/jobs/{id}", h.GetJob)

	// Stats
	mux.HandleFunc("GET /api/stats", h.Stats)

	// Apply middleware
	var chain http.Handler = mux
	chain = middleware.MaxBodySize(10 << 20)(chain)
	chain = rateLimiter.Limit(chain)
	chain = middleware.SecurityHeaders(chain)

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      chain,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Printf("PixelForge starting on port %s", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	<-done
	log.Println("Shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Shutdown failed: %v", err)
	}

	log.Println("Server stopped")
}
