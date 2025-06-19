package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	coreConfig "github.com/metfin/core/config"
	"github.com/metfin/core/database"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/wnt/mercon/internal/config"
	"github.com/wnt/mercon/internal/logger"
	"github.com/wnt/mercon/internal/queue"
	"github.com/wnt/mercon/internal/rpc"
	"github.com/wnt/mercon/internal/worker"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	log := logger.New(cfg.LogLevel)
	log.Info().Msg("Starting Mercon data scraper")

	// Initialize databases
	log.Info().Msg("Initializing database connections")
	
	// Initialize chain database (for raw transactions)
	chainDBConfig := &coreConfig.DatabaseConfig{
		Host:     cfg.ChainDBHost,
		User:     cfg.ChainDBUser,
		Password: cfg.ChainDBPassword,
		DBName:   cfg.ChainDBName,
		Port:     cfg.ChainDBPort,
		SSLMode:  cfg.ChainDBSSLMode,
	}
	
	database.InitDB(chainDBConfig)
	
	// Run raw chain migrations
	if err := database.RunRawMigrations(); err != nil {
		log.Error().Err(err).Msg("Failed to run raw chain migrations")
		// Continue anyway as this is expected in development
	}

	// Initialize Redis queue
	log.Info().Str("redis_url", cfg.RedisURL).Msg("Connecting to Redis queue")
	queueClient, err := queue.NewClient(cfg.RedisURL, log)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to Redis")
	}
	defer queueClient.Close()

	// Initialize RPC pool
	log.Info().Int("endpoints", len(cfg.RPCEndpoints)).Msg("Initializing RPC pool")
	rpcPool := rpc.NewPool(cfg.RPCEndpoints, log)

	// Initialize worker manager
	log.Info().Msg("Initializing worker manager")
	workerManager := worker.NewManager(cfg, queueClient, rpcPool, log)

	// Initialize metrics HTTP server
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ready"))
	})
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("healthy"))
	})
	mux.HandleFunc("/stats", func(w http.ResponseWriter, r *http.Request) {
		stats := workerManager.GetStats()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "%+v", stats)
	})

	server := &http.Server{
		Addr:    ":" + cfg.MetricsPort,
		Handler: mux,
	}

	// Start metrics server in background
	go func() {
		log.Info().Str("port", cfg.MetricsPort).Msg("Starting metrics server")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("Failed to start metrics server")
		}
	}()

	// Start worker manager
	log.Info().Msg("Starting worker manager")
	if err := workerManager.Start(); err != nil {
		log.Fatal().Err(err).Msg("Failed to start worker manager")
	}

	log.Info().
		Int("min_workers", cfg.MinWorkers).
		Int("max_workers", cfg.MaxWorkers).
		Int("rpc_endpoints", len(cfg.RPCEndpoints)).
		Msg("Mercon started successfully")

	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Wait for shutdown signal
	<-sigChan
	log.Info().Msg("Received shutdown signal, starting graceful shutdown")

	// Stop worker manager first
	if err := workerManager.Stop(); err != nil {
		log.Error().Err(err).Msg("Error stopping worker manager")
	}

	// Shutdown metrics server
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Error().Err(err).Msg("Failed to shutdown metrics server gracefully")
	}

	// Close database connections
	if err := database.CloseDB(); err != nil {
		log.Error().Err(err).Msg("Error closing database connections")
	}

	log.Info().Msg("Mercon shutdown complete")
}
