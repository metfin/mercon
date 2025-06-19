package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	// Save original env vars
	originalVars := map[string]string{
		"REDIS_URL":         os.Getenv("REDIS_URL"),
		"PG_CHAIN_DSN":      os.Getenv("PG_CHAIN_DSN"),
		"PG_ANALYTICS_DSN":  os.Getenv("PG_ANALYTICS_DSN"),
		"RPC_ENDPOINTS":     os.Getenv("RPC_ENDPOINTS"),
		"MIN_WORKERS":       os.Getenv("MIN_WORKERS"),
		"MAX_WORKERS":       os.Getenv("MAX_WORKERS"),
		"LOG_LEVEL":         os.Getenv("LOG_LEVEL"),
		"POSTHOG_KEY":       os.Getenv("POSTHOG_KEY"),
		"METRICS_PORT":      os.Getenv("METRICS_PORT"),
	}

	// Restore env vars after test
	defer func() {
		for key, value := range originalVars {
			if value == "" {
				os.Unsetenv(key)
			} else {
				os.Setenv(key, value)
			}
		}
	}()

	t.Run("successful load with all required vars", func(t *testing.T) {
		// Set required environment variables
		os.Setenv("REDIS_URL", "redis://localhost:6379")
		os.Setenv("PG_CHAIN_DSN", "postgres://user:pass@localhost/chain")
		os.Setenv("PG_ANALYTICS_DSN", "postgres://user:pass@localhost/analytics")
		os.Setenv("RPC_ENDPOINTS", "https://api.mainnet-beta.solana.com,https://rpc.ankr.com/solana")
		os.Setenv("MIN_WORKERS", "2")
		os.Setenv("MAX_WORKERS", "10")
		os.Setenv("LOG_LEVEL", "debug")
		os.Setenv("POSTHOG_KEY", "test_key")
		os.Setenv("METRICS_PORT", "9090")

		cfg, err := Load()
		require.NoError(t, err)

		assert.Equal(t, "redis://localhost:6379", cfg.RedisURL)
		assert.Equal(t, "postgres://user:pass@localhost/chain", cfg.PGChainDSN)
		assert.Equal(t, "postgres://user:pass@localhost/analytics", cfg.PGAnalyticsDSN)
		assert.Equal(t, []string{"https://api.mainnet-beta.solana.com", "https://rpc.ankr.com/solana"}, cfg.RPCEndpoints)
		assert.Equal(t, 2, cfg.MinWorkers)
		assert.Equal(t, 10, cfg.MaxWorkers)
		assert.Equal(t, "debug", cfg.LogLevel)
		assert.Equal(t, "test_key", cfg.PosthogKey)
		assert.Equal(t, "9090", cfg.MetricsPort)
	})

	t.Run("missing required environment variables", func(t *testing.T) {
		// Clear required env vars
		os.Unsetenv("REDIS_URL")
		os.Unsetenv("PG_CHAIN_DSN")
		os.Unsetenv("PG_ANALYTICS_DSN")
		os.Unsetenv("RPC_ENDPOINTS")

		_, err := Load()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "RPC_ENDPOINTS environment variable is required")
	})

	t.Run("invalid worker configuration", func(t *testing.T) {
		// Set valid required vars but invalid worker config
		os.Setenv("REDIS_URL", "redis://localhost:6379")
		os.Setenv("PG_CHAIN_DSN", "postgres://user:pass@localhost/chain")
		os.Setenv("PG_ANALYTICS_DSN", "postgres://user:pass@localhost/analytics")
		os.Setenv("RPC_ENDPOINTS", "https://api.mainnet-beta.solana.com")
		os.Setenv("MIN_WORKERS", "10")
		os.Setenv("MAX_WORKERS", "5") // Max less than min

		_, err := Load()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "MAX_WORKERS must be greater than or equal to MIN_WORKERS")
	})

	t.Run("invalid log level", func(t *testing.T) {
		// Set valid required vars but invalid log level
		os.Setenv("REDIS_URL", "redis://localhost:6379")
		os.Setenv("PG_CHAIN_DSN", "postgres://user:pass@localhost/chain")
		os.Setenv("PG_ANALYTICS_DSN", "postgres://user:pass@localhost/analytics")
		os.Setenv("RPC_ENDPOINTS", "https://api.mainnet-beta.solana.com")
		os.Setenv("MIN_WORKERS", "4")
		os.Setenv("MAX_WORKERS", "50")
		os.Setenv("LOG_LEVEL", "invalid")

		_, err := Load()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid LOG_LEVEL")
	})

	t.Run("defaults are applied", func(t *testing.T) {
		// Set only required vars, test defaults
		os.Setenv("PG_CHAIN_DSN", "postgres://user:pass@localhost/chain")
		os.Setenv("PG_ANALYTICS_DSN", "postgres://user:pass@localhost/analytics")
		os.Setenv("RPC_ENDPOINTS", "https://api.mainnet-beta.solana.com")
		os.Unsetenv("REDIS_URL")
		os.Unsetenv("MIN_WORKERS")
		os.Unsetenv("MAX_WORKERS")
		os.Unsetenv("LOG_LEVEL")
		os.Unsetenv("METRICS_PORT")

		cfg, err := Load()
		require.NoError(t, err)

		assert.Equal(t, "redis://localhost:6379", cfg.RedisURL)
		assert.Equal(t, 4, cfg.MinWorkers)
		assert.Equal(t, 50, cfg.MaxWorkers)
		assert.Equal(t, "info", cfg.LogLevel)
		assert.Equal(t, "9100", cfg.MetricsPort)
	})
} 