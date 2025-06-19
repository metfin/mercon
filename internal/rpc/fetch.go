package rpc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/rs/zerolog"
	"github.com/wnt/mercon/internal/metrics"
)

// RpcTransaction represents a raw Solana transaction from the RPC
type RpcTransaction struct {
	Slot        uint64                 `json:"slot"`
	Transaction map[string]interface{} `json:"transaction"`
	Meta        map[string]interface{} `json:"meta"`
	BlockTime   *int64                 `json:"blockTime"`
}

// RpcRequest represents a JSON RPC request
type RpcRequest struct {
	Jsonrpc string        `json:"jsonrpc"`
	ID      string        `json:"id"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
}

// RpcResponse represents a JSON RPC response
type RpcResponse struct {
	Jsonrpc string      `json:"jsonrpc"`
	ID      string      `json:"id"`
	Result  interface{} `json:"result"`
	Error   *RpcError   `json:"error"`
}

// RpcError represents an RPC error
type RpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Fetcher handles RPC transaction fetching with retries and backoff
type Fetcher struct {
	pool   *Pool
	logger zerolog.Logger
}

// NewFetcher creates a new transaction fetcher
func NewFetcher(pool *Pool, logger zerolog.Logger) *Fetcher {
	return &Fetcher{
		pool:   pool,
		logger: logger.With().Str("component", "rpc_fetcher").Logger(),
	}
}

// FetchTransaction fetches a transaction by signature with retry logic
func (f *Fetcher) FetchTransaction(ctx context.Context, signature string) (*RpcTransaction, error) {
	const maxRetries = 5
	baseDelay := 250 * time.Millisecond
	
	for attempt := 0; attempt <= maxRetries; attempt++ {
		tx, err := f.fetchTransactionOnce(ctx, signature)
		if err == nil {
			metrics.RecordRPCRequest("success")
			return tx, nil
		}
		
		// Log the error
		f.logger.Warn().
			Err(err).
			Str("signature", signature).
			Int("attempt", attempt+1).
			Int("max_retries", maxRetries).
			Msg("Failed to fetch transaction")
		
		// Check if we should retry
		if attempt == maxRetries {
			metrics.RecordRPCRequest("failed")
			return nil, fmt.Errorf("failed to fetch transaction after %d attempts: %w", maxRetries+1, err)
		}
		
		// Exponential backoff with jitter
		delay := baseDelay * time.Duration(1<<attempt)
		if delay > 30*time.Second {
			delay = 30 * time.Second
		}
		
		f.logger.Debug().
			Str("signature", signature).
			Dur("delay", delay).
			Msg("Retrying transaction fetch after delay")
		
		select {
		case <-time.After(delay):
			// Continue to next attempt
		case <-ctx.Done():
			metrics.RecordRPCRequest("cancelled")
			return nil, ctx.Err()
		}
	}
	
	return nil, fmt.Errorf("unreachable code")
}

// fetchTransactionOnce performs a single transaction fetch attempt
func (f *Fetcher) fetchTransactionOnce(ctx context.Context, signature string) (*RpcTransaction, error) {
	client, endpoint, err := f.pool.GetClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get RPC client: %w", err)
	}
	
	// Create RPC request
	request := RpcRequest{
		Jsonrpc: "2.0",
		ID:      "1",
		Method:  "getTransaction",
		Params: []interface{}{
			signature,
			map[string]interface{}{
				"encoding":                       "json",
				"commitment":                     "confirmed",
				"maxSupportedTransactionVersion": 0,
			},
		},
	}
	
	requestBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal RPC request: %w", err)
	}
	
	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}
	
	httpReq.Header.Set("Content-Type", "application/json")
	
	// Make the request
	startTime := time.Now()
	resp, err := client.Do(httpReq)
	duration := time.Since(startTime)
	
	if err != nil {
		f.handleError(endpoint, err, duration)
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()
	
	// Handle HTTP status codes
	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode == http.StatusServiceUnavailable {
		f.handleRateLimit(endpoint)
		return nil, fmt.Errorf("rate limited by endpoint %s: status %d", endpoint, resp.StatusCode)
	}
	
	if resp.StatusCode != http.StatusOK {
		f.pool.MarkUnhealthy(endpoint)
		return nil, fmt.Errorf("unexpected status code from %s: %d", endpoint, resp.StatusCode)
	}
	
	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}
	
	// Parse RPC response
	var rpcResponse RpcResponse
	if err := json.Unmarshal(body, &rpcResponse); err != nil {
		return nil, fmt.Errorf("failed to unmarshal RPC response: %w", err)
	}
	
	// Check for RPC errors
	if rpcResponse.Error != nil {
		return nil, fmt.Errorf("RPC error from %s: code %d, message: %s", 
			endpoint, rpcResponse.Error.Code, rpcResponse.Error.Message)
	}
	
	// Check if transaction was found
	if rpcResponse.Result == nil {
		return nil, fmt.Errorf("transaction not found: %s", signature)
	}
	
	// Parse the transaction result
	resultBytes, err := json.Marshal(rpcResponse.Result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal transaction result: %w", err)
	}
	
	var transaction RpcTransaction
	if err := json.Unmarshal(resultBytes, &transaction); err != nil {
		return nil, fmt.Errorf("failed to unmarshal transaction: %w", err)
	}
	
	f.logger.Debug().
		Str("signature", signature).
		Str("endpoint", endpoint).
		Dur("duration", duration).
		Msg("Successfully fetched transaction")
	
	// Mark endpoint as healthy since request succeeded
	f.pool.MarkHealthy(endpoint)
	
	return &transaction, nil
}

// FetchSignatures fetches transaction signatures for a wallet
func (f *Fetcher) FetchSignatures(ctx context.Context, wallet string, before string, limit int) ([]string, error) {
	client, endpoint, err := f.pool.GetClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get RPC client: %w", err)
	}
	
	params := []interface{}{
		wallet,
		map[string]interface{}{
			"limit":      limit,
			"commitment": "confirmed",
		},
	}
	
	if before != "" {
		params[1].(map[string]interface{})["before"] = before
	}
	
	request := RpcRequest{
		Jsonrpc: "2.0",
		ID:      "1",
		Method:  "getSignaturesForAddress",
		Params:  params,
	}
	
	requestBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal RPC request: %w", err)
	}
	
	httpReq, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}
	
	httpReq.Header.Set("Content-Type", "application/json")
	
	startTime := time.Now()
	resp, err := client.Do(httpReq)
	duration := time.Since(startTime)
	
	if err != nil {
		f.handleError(endpoint, err, duration)
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode == http.StatusServiceUnavailable {
		f.handleRateLimit(endpoint)
		return nil, fmt.Errorf("rate limited by endpoint %s: status %d", endpoint, resp.StatusCode)
	}
	
	if resp.StatusCode != http.StatusOK {
		f.pool.MarkUnhealthy(endpoint)
		return nil, fmt.Errorf("unexpected status code from %s: %d", endpoint, resp.StatusCode)
	}
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}
	
	var rpcResponse RpcResponse
	if err := json.Unmarshal(body, &rpcResponse); err != nil {
		return nil, fmt.Errorf("failed to unmarshal RPC response: %w", err)
	}
	
	if rpcResponse.Error != nil {
		return nil, fmt.Errorf("RPC error from %s: code %d, message: %s", 
			endpoint, rpcResponse.Error.Code, rpcResponse.Error.Message)
	}
	
	// Parse signature results
	signatures := make([]string, 0)
	if resultSlice, ok := rpcResponse.Result.([]interface{}); ok {
		for _, item := range resultSlice {
			if sigMap, ok := item.(map[string]interface{}); ok {
				if sig, ok := sigMap["signature"].(string); ok {
					signatures = append(signatures, sig)
				}
			}
		}
	}
	
	f.logger.Debug().
		Str("wallet", wallet).
		Str("endpoint", endpoint).
		Int("signatures", len(signatures)).
		Dur("duration", duration).
		Msg("Successfully fetched signatures")
	
	metrics.RecordRPCRequest("success")
	f.pool.MarkHealthy(endpoint)
	
	return signatures, nil
}

// handleError handles RPC errors and marks endpoints as unhealthy if needed
func (f *Fetcher) handleError(endpoint string, err error, duration time.Duration) {
	f.logger.Error().
		Err(err).
		Str("endpoint", endpoint).
		Dur("duration", duration).
		Msg("RPC request failed")
	
	// Mark endpoint as unhealthy on network errors
	f.pool.MarkUnhealthy(endpoint)
	metrics.RecordRPCRequest("error")
}

// handleRateLimit handles rate limiting by setting cooldown
func (f *Fetcher) handleRateLimit(endpoint string) {
	f.logger.Warn().
		Str("endpoint", endpoint).
		Msg("Rate limited by endpoint")
	
	// Set 5-minute cooldown for rate limited endpoints
	f.pool.SetCooldown(endpoint, 5*time.Minute)
	metrics.RecordRPCRequest("rate_limited")
} 