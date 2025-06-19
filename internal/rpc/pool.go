package rpc

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/wnt/mercon/internal/metrics"
	"golang.org/x/time/rate"
)

// Pool manages a pool of RPC endpoints with load balancing and rate limiting
type Pool struct {
	endpoints []*Endpoint
	current   int
	mutex     sync.RWMutex
	logger    zerolog.Logger
}

// Endpoint represents a single RPC endpoint with its own rate limiter
type Endpoint struct {
	URL        string
	client     *http.Client
	limiter    *rate.Limiter
	healthy    bool
	cooldownUntil time.Time
	mutex      sync.RWMutex
}

// NewPool creates a new RPC pool with the given endpoints
func NewPool(urls []string, logger zerolog.Logger) *Pool {
	endpoints := make([]*Endpoint, len(urls))
	
	for i, url := range urls {
		endpoints[i] = &Endpoint{
			URL: url,
			client: &http.Client{
				Timeout: 30 * time.Second,
			},
			// Rate limit to ~2 req/s per endpoint to stay under free tier limits
			limiter: rate.NewLimiter(rate.Limit(2.0), 5),
			healthy: true,
		}
		
		// Set initial health status in metrics
		metrics.SetRPCEndpointHealth(url, true)
	}
	
	return &Pool{
		endpoints: endpoints,
		current:   rand.Intn(len(endpoints)),
		logger:    logger.With().Str("component", "rpc_pool").Logger(),
	}
}

// GetClient returns the next available RPC client using round-robin
func (p *Pool) GetClient(ctx context.Context) (*http.Client, string, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	
	attempts := 0
	startIndex := p.current
	
	for {
		endpoint := p.endpoints[p.current]
		p.current = (p.current + 1) % len(p.endpoints)
		attempts++
		
		// Check if endpoint is in cooldown
		endpoint.mutex.RLock()
		inCooldown := time.Now().Before(endpoint.cooldownUntil)
		healthy := endpoint.healthy
		endpoint.mutex.RUnlock()
		
		if inCooldown {
			p.logger.Debug().
				Str("endpoint", endpoint.URL).
				Time("cooldown_until", endpoint.cooldownUntil).
				Msg("Endpoint in cooldown, skipping")
			
			// If we've tried all endpoints, continue to rate limiting check
			if attempts >= len(p.endpoints) {
				break
			}
			continue
		}
		
		if !healthy {
			p.logger.Debug().
				Str("endpoint", endpoint.URL).
				Msg("Endpoint unhealthy, skipping")
			
			// If we've tried all endpoints, continue to rate limiting check
			if attempts >= len(p.endpoints) {
				break
			}
			continue
		}
		
		// Check rate limit
		if endpoint.limiter.Allow() {
			p.logger.Debug().
				Str("endpoint", endpoint.URL).
				Msg("Selected RPC endpoint")
			return endpoint.client, endpoint.URL, nil
		}
		
		p.logger.Debug().
			Str("endpoint", endpoint.URL).
			Msg("Endpoint rate limited, trying next")
		
		// If we've tried all endpoints, break
		if attempts >= len(p.endpoints) {
			break
		}
	}
	
	// All endpoints are rate limited or unhealthy, wait for the first available one
	endpoint := p.endpoints[startIndex]
	
	p.logger.Debug().
		Str("endpoint", endpoint.URL).
		Msg("All endpoints rate limited, waiting for availability")
	
	// Wait for rate limit to reset with context cancellation
	reservation := endpoint.limiter.Reserve()
	if !reservation.OK() {
		return nil, "", fmt.Errorf("rate limiter failed to make reservation")
	}
	
	delay := reservation.Delay()
	if delay > 0 {
		select {
		case <-time.After(delay):
			// Rate limit delay completed
		case <-ctx.Done():
			reservation.Cancel()
			return nil, "", ctx.Err()
		}
	}
	
	return endpoint.client, endpoint.URL, nil
}

// MarkUnhealthy marks an endpoint as unhealthy
func (p *Pool) MarkUnhealthy(url string) {
	for _, endpoint := range p.endpoints {
		if endpoint.URL == url {
			endpoint.mutex.Lock()
			endpoint.healthy = false
			endpoint.mutex.Unlock()
			
			metrics.SetRPCEndpointHealth(url, false)
			p.logger.Warn().Str("endpoint", url).Msg("Marked endpoint as unhealthy")
			break
		}
	}
}

// MarkHealthy marks an endpoint as healthy
func (p *Pool) MarkHealthy(url string) {
	for _, endpoint := range p.endpoints {
		if endpoint.URL == url {
			endpoint.mutex.Lock()
			endpoint.healthy = true
			endpoint.cooldownUntil = time.Time{} // Clear cooldown
			endpoint.mutex.Unlock()
			
			metrics.SetRPCEndpointHealth(url, true)
			p.logger.Info().Str("endpoint", url).Msg("Marked endpoint as healthy")
			break
		}
	}
}

// SetCooldown puts an endpoint in cooldown for the specified duration
func (p *Pool) SetCooldown(url string, duration time.Duration) {
	for _, endpoint := range p.endpoints {
		if endpoint.URL == url {
			endpoint.mutex.Lock()
			endpoint.cooldownUntil = time.Now().Add(duration)
			endpoint.mutex.Unlock()
			
			p.logger.Warn().
				Str("endpoint", url).
				Dur("duration", duration).
				Msg("Set endpoint cooldown")
			break
		}
	}
}

// GetHealthyEndpointCount returns the number of healthy endpoints
func (p *Pool) GetHealthyEndpointCount() int {
	count := 0
	for _, endpoint := range p.endpoints {
		endpoint.mutex.RLock()
		if endpoint.healthy && time.Now().After(endpoint.cooldownUntil) {
			count++
		}
		endpoint.mutex.RUnlock()
	}
	return count
}

// GetStats returns pool statistics
func (p *Pool) GetStats() map[string]interface{} {
	stats := map[string]interface{}{
		"total_endpoints":   len(p.endpoints),
		"healthy_endpoints": p.GetHealthyEndpointCount(),
		"endpoints":         make([]map[string]interface{}, len(p.endpoints)),
	}
	
	for i, endpoint := range p.endpoints {
		endpoint.mutex.RLock()
		endpointStats := map[string]interface{}{
			"url":             endpoint.URL,
			"healthy":         endpoint.healthy,
			"in_cooldown":     time.Now().Before(endpoint.cooldownUntil),
			"cooldown_until":  endpoint.cooldownUntil,
		}
		endpoint.mutex.RUnlock()
		
		stats["endpoints"].([]map[string]interface{})[i] = endpointStats
	}
	
	return stats
} 