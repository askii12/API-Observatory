package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/askii12/api-observatory/internal/storage/postgres"
	httptransport "github.com/askii12/api-observatory/internal/transport/http"
	"github.com/askii12/api-observatory/migrations"
)

func main() {
	addr := env("HTTP_ADDR", ":8080")
	dsn := os.Getenv("DATABASE_URL")

	if dsn == "" {
		log.Fatal("DATABASE_URL is required")
	}

	ctx, cancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer cancel()

	db, err := postgres.Open(ctx, dsn)
	if err != nil {
		log.Fatalf("database connection failed: %v", err)
	}
	defer db.Close()

	if err := migrations.Run(ctx, db); err != nil {
		log.Fatalf("database migrations failed: %v", err)
	}

	log.Println("Database connected")
	log.Println("Database migrations completed")

	// Initialize repository and handler.
	monitorRepo := postgres.NewMonitorRepository(db)
	monitorHandler := httptransport.NewMonitorHandler(monitorRepo)

	mux := http.NewServeMux()

	// Health endpoints.
	mux.HandleFunc("GET /healthz", func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	mux.HandleFunc("GET /readyz", func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		if err := db.PingContext(r.Context()); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(`{"status":"not_ready"}`))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ready"}`))
	})

	// Monitor endpoints.
	mux.HandleFunc("POST /api/v1/monitors", monitorHandler.Create)
	mux.HandleFunc("GET /api/v1/monitors", monitorHandler.List)
	mux.HandleFunc("GET /api/v1/monitors/{id}", monitorHandler.Get)
	mux.HandleFunc("PUT /api/v1/monitors/{id}", monitorHandler.Update)
	mux.HandleFunc("DELETE /api/v1/monitors/{id}", monitorHandler.Delete)

	server := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("API Observatory listening on %s", addr)

	if err := server.ListenAndServe(); err != nil &&
		err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return fallback
}
