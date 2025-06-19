package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/wnt/mercon/internal/config"
	"github.com/wnt/mercon/internal/logger"
	"github.com/wnt/mercon/internal/metrics"
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

	// TODO: Initialize and start worker manager (Phase 1)
	log.Info().
		Int("min_workers", cfg.MinWorkers).
		Int("max_workers", cfg.MaxWorkers).
		Int("rpc_endpoints", len(cfg.RPCEndpoints)).
		Msg("Configuration loaded successfully")

	// Set initial metrics
	metrics.WorkersActive.Set(0)
	metrics.WalletQueueLength.Set(0)

	// Setup graceful shutdown

	// Listen for interrupt signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Wait for shutdown signal
	<-sigChan
	log.Info().Msg("Received shutdown signal, starting graceful shutdown")

	// Shutdown metrics server
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Error().Err(err).Msg("Failed to shutdown metrics server gracefully")
	}

	log.Info().Msg("Mercon shutdown complete")
}
