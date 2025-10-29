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
	"github.com/xGuthub/metrics-collection-service/internal/config"
	"github.com/xGuthub/metrics-collection-service/internal/handler"
	"github.com/xGuthub/metrics-collection-service/internal/logger"
	"github.com/xGuthub/metrics-collection-service/internal/repository"
	"github.com/xGuthub/metrics-collection-service/internal/service"
)

func main() {
	if err := logger.Initialize("INFO"); err != nil {
		log.Fatalf("failed to init logger: %v", err)
	}

	srvCfg, err := config.LoadServerConfigFromFlags()
	if err != nil {
		logger.Log.Fatalf("failed to parse flags: %v", err)
	}

	// Choose storage: PostgreSQL if DSN provided, otherwise in-memory
	var storage service.Storage = repository.NewMemStorage()
	var pgStorage *repository.PostgresStorage
	if srvCfg.DatabaseDSN != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		s, err := repository.NewPostgresStorage(ctx, srvCfg.DatabaseDSN)
		if err != nil {
			logger.Log.Fatalf("failed to connect postgres: %v", err)
		}
		pgStorage = s
		// Use DB-backed storage
		storage = s
		logger.Log.Infof("connected to PostgreSQL")
	}
	metricsService := service.NewMetricsService(storage)
	metricsService.SetStateStore(repository.NewFileStateStore())
	// Configure persistence based on server config
	metricsService.ConfigurePersistence(service.PersistenceConfig{
		FilePath:      srvCfg.FileStoragePath,
		StoreInterval: srvCfg.StoreIntervale,
		Restore:       srvCfg.Restore,
	})

	// Restore state on start if enabled
	if err := metricsService.RestoreState(); err != nil {
		logger.Log.Errorf("failed to restore metrics: %v", err)
	}
	metricsHandler := handler.NewMetricsHandler(metricsService)

	r := chi.NewRouter()
	r.Use(WithLogging)
	r.Use(WithGzip)

	// Health check endpoint that verifies DB connectivity
	var dbPing handler.DBPinger
	if pgStorage != nil {
		dbPing = pgStorage
	}
	healthHandler := handler.NewHealthHandler(dbPing)
	r.Get("/ping", healthHandler.PingHandler)
	r.Get("/", metricsHandler.HomeHandler)
	r.Post("/update/", metricsHandler.UpdateJSONHandler)
	r.Post("/update/*", metricsHandler.UpdateHandler)
	r.Post("/value/", metricsHandler.ValueJSONHandler)
	r.Get("/value/*", metricsHandler.ValueHandler)

	server := &http.Server{
		Addr:              srvCfg.Address,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Start periodic autosave if configured
	metricsService.StartAutoSave(ctx, func(err error) {
		logger.Log.Errorf("autosave error: %v", err)
	})

	go func() {
		logger.Log.Infof("metrics server listening on http://%s", server.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Log.Fatalf("server error: %v", err)
		}
	}()

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Log.Infof("graceful shutdown failed: %v", err)
		_ = server.Close()
	}

	// Ensure all accumulated metrics are saved on normal shutdown.
	if err := metricsService.SaveState(); err != nil {
		logger.Log.Errorf("failed to save metrics on shutdown: %v", err)
	}
	if pgStorage != nil {
		_ = pgStorage.Close()
	}
	logger.Log.Infof("server stopped")
}
