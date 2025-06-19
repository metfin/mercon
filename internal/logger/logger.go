package logger

import (
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// New creates and configures a new zerolog logger
func New(logLevel string) zerolog.Logger {
	// Set global log level
	level, err := zerolog.ParseLevel(strings.ToLower(logLevel))
	if err != nil {
		level = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(level)

	// Configure console writer for human-readable output in development
	if os.Getenv("API_ENV") == "development" {
		log.Logger = log.Output(zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: time.RFC3339,
		})
	}

	// Create structured logger with common fields
	logger := zerolog.New(os.Stdout).
		Level(level).
		With().
		Timestamp().
		Str("service", "mercon").
		Logger()

	return logger
}

// WithWorker adds worker ID to logger context
func WithWorker(logger zerolog.Logger, workerID string) zerolog.Logger {
	return logger.With().Str("worker_id", workerID).Logger()
}

// WithWallet adds wallet address to logger context
func WithWallet(logger zerolog.Logger, wallet string) zerolog.Logger {
	return logger.With().Str("wallet", wallet).Logger()
}

// WithRPCEndpoint adds RPC endpoint to logger context
func WithRPCEndpoint(logger zerolog.Logger, endpoint string) zerolog.Logger {
	return logger.With().Str("rpc_endpoint", endpoint).Logger()
} 