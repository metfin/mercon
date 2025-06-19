package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config holds all configuration for Mercon
type Config struct {
	// Redis configuration
	RedisURL string

	// Database configuration
	ChainDBName     string
	ChainDBHost       string
	ChainDBUser       string
	ChainDBPassword   string
	ChainDBPort       string
	ChainDBSSLMode    string

	AnalyticsDBName     string
	AnalyticsDBHost       string
	AnalyticsDBUser       string
	AnalyticsDBPassword   string
	AnalyticsDBPort       string
	AnalyticsDBSSLMode    string

	// RPC configuration
	RPCEndpoints []string

	// Worker configuration
	MinWorkers int
	MaxWorkers int

	// Logging configuration
	LogLevel string

	// Analytics configuration
	PosthogKey string

	// Metrics configuration
	MetricsPort string
}

// Load reads configuration from environment variables and validates it
func Load() (Config, error) {
	cfg := Config{
		RedisURL:       getEnv("REDIS_URL", "redis://localhost:6379"),
		ChainDBName:     getEnv("CHAIN_DB_NAME", ""),
		ChainDBHost:     getEnv("CHAIN_DB_HOST", ""),
		ChainDBUser:     getEnv("CHAIN_DB_USER", ""),
		ChainDBPassword: getEnv("CHAIN_DB_PASSWORD", ""),
		ChainDBPort:     getEnv("CHAIN_DB_PORT", ""),
		ChainDBSSLMode:  getEnv("CHAIN_DB_SSL_MODE", ""),
		AnalyticsDBName: getEnv("ANALYTICS_DB_NAME", ""),
		AnalyticsDBHost: getEnv("ANALYTICS_DB_HOST", ""),
		AnalyticsDBUser: getEnv("ANALYTICS_DB_USER", ""),
		AnalyticsDBPassword: getEnv("ANALYTICS_DB_PASSWORD", ""),
		AnalyticsDBPort: getEnv("ANALYTICS_DB_PORT", ""),
		AnalyticsDBSSLMode: getEnv("ANALYTICS_DB_SSL_MODE", ""),
		LogLevel:       getEnv("LOG_LEVEL", "info"),
		PosthogKey:     getEnv("POSTHOG_KEY", ""),
		MetricsPort:    getEnv("METRICS_PORT", "9100"),
	}

	// Parse RPC endpoints
	rpcEndpointsStr := getEnv("RPC_ENDPOINTS", "")
	if rpcEndpointsStr == "" {
		return cfg, fmt.Errorf("RPC_ENDPOINTS environment variable is required")
	}
	cfg.RPCEndpoints = strings.Split(rpcEndpointsStr, ",")
	for i, endpoint := range cfg.RPCEndpoints {
		cfg.RPCEndpoints[i] = strings.TrimSpace(endpoint)
	}

	// Parse worker configuration
	var err error
	cfg.MinWorkers, err = parseIntEnv("MIN_WORKERS", 4)
	if err != nil {
		return cfg, fmt.Errorf("invalid MIN_WORKERS: %w", err)
	}

	cfg.MaxWorkers, err = parseIntEnv("MAX_WORKERS", 50)
	if err != nil {
		return cfg, fmt.Errorf("invalid MAX_WORKERS: %w", err)
	}

	// Validate configuration
	if err := cfg.validate(); err != nil {
		return cfg, fmt.Errorf("configuration validation failed: %w", err)
	}

	return cfg, nil
}

// validate checks that the configuration is valid
func (c Config) validate() error {
	if c.RedisURL == "" {
		return fmt.Errorf("REDIS_URL is required")
	}

	if c.ChainDBName == "" {
		return fmt.Errorf("CHAIN_DB_NAME is required")
	}

	if c.AnalyticsDBName == "" {
		return fmt.Errorf("ANALYTICS_DB_NAME is required")
	}

	if len(c.RPCEndpoints) == 0 {
		return fmt.Errorf("at least one RPC endpoint is required")
	}

	if c.MinWorkers < 1 {
		return fmt.Errorf("MIN_WORKERS must be at least 1")
	}

	if c.MaxWorkers < c.MinWorkers {
		return fmt.Errorf("MAX_WORKERS must be greater than or equal to MIN_WORKERS")
	}

	validLogLevels := map[string]bool{
		"trace": true,
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
		"fatal": true,
		"panic": true,
	}

	if !validLogLevels[strings.ToLower(c.LogLevel)] {
		return fmt.Errorf("invalid LOG_LEVEL: %s (must be one of: trace, debug, info, warn, error, fatal, panic)", c.LogLevel)
	}

	return nil
}

// getEnv retrieves an environment variable with a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// parseIntEnv parses an integer environment variable with a default value
func parseIntEnv(key string, defaultValue int) (int, error) {
	str := os.Getenv(key)
	if str == "" {
		return defaultValue, nil
	}
	return strconv.Atoi(str)
} 