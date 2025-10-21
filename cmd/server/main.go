package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/xGuthub/metrics-collection-service/internal/handler"
	"github.com/xGuthub/metrics-collection-service/internal/repository"
	"github.com/xGuthub/metrics-collection-service/internal/service"
)

func main() {
	storage := repository.NewMemStorage()
	metricsService := service.NewMetricsService(storage)
	metricsHandler := handler.NewMetricsHandler(metricsService)
	srv := NewServer(metricsHandler)

	r := chi.NewRouter()
	r.Post("/update/*", http.HandlerFunc(srv.serveUpdate))
	r.Get("/value/*", http.HandlerFunc(srv.serveGet))

	server := &http.Server{
		Addr:              "localhost:8080",
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Printf("metrics server listening on http://%s", server.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server error: %v", err)
		}
	}()

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("graceful shutdown failed: %v", err)
		_ = server.Close()
	}
	log.Printf("server stopped")
}
