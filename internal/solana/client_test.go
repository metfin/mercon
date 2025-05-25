package solana

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/wnt/mercon/internal/utils"
)

func TestLoadConfigFromEnv(t *testing.T) {
	// Save original environment and restore after test
	origRPCURL := os.Getenv("RPC_URL")
	origTimeout := os.Getenv("RPC_TIMEOUT")
	defer func() {
		os.Setenv("RPC_URL", origRPCURL)
		os.Setenv("RPC_TIMEOUT", origTimeout)
	}()

	// Test case: missing RPC_URL
	os.Unsetenv("RPC_URL")
	_, err := loadConfigFromEnv()
	if err != ErrMissingRPCURL {
		t.Errorf("Expected ErrMissingRPCURL, got %v", err)
	}

	// Test case: with RPC_URL
	os.Setenv("RPC_URL", "https://example.com")
	config, err := loadConfigFromEnv()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if config.RPCURL != "https://example.com" {
		t.Errorf("Expected RPCURL to be https://example.com, got %s", config.RPCURL)
	}
	if config.Timeout != DefaultTimeout {
		t.Errorf("Expected default timeout, got %v", config.Timeout)
	}

	// Test case: with custom timeout
	os.Setenv("RPC_TIMEOUT", "60s")
	config, err = loadConfigFromEnv()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if config.Timeout != 60*time.Second {
		t.Errorf("Expected timeout to be 60s, got %v", config.Timeout)
	}
}

func TestGetTransactions(t *testing.T) {
	// Save original environment and restore after test
	origAPIKey := os.Getenv("HELIUS_API_KEY")
	defer os.Setenv("HELIUS_API_KEY", origAPIKey)

	// Test case: missing API key
	os.Unsetenv("HELIUS_API_KEY")
	client := &Client{
		httpClient: utils.NewHTTPClient(),
	}
	_, err := client.GetTransactions(context.Background(), "someAddress", Filters{})
	if err != ErrMissingAPIKey {
		t.Errorf("Expected ErrMissingAPIKey, got %v", err)
	}

	// Test case: with API key and successful response
	os.Setenv("HELIUS_API_KEY", "testApiKey")

	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request parameters
		if r.URL.Path != "/addresses/testAddress/transactions" {
			t.Errorf("Expected path /addresses/testAddress/transactions, got %s", r.URL.Path)
		}
		if apiKey := r.URL.Query().Get("api-key"); apiKey != "testApiKey" {
			t.Errorf("Expected api-key=testApiKey, got %s", apiKey)
		}

		// Return a mock response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[
			{
				"signature": "testSignature",
				"description": "Test Transaction",
				"type": "TRANSFER",
				"source": "SYSTEM_PROGRAM",
				"fee": 5000,
				"feePayer": "testFeePayer",
				"slot": 12345678,
				"timestamp": 1667289600
			}
		]`))
	}))
	defer server.Close()

	// Update client to use test server
	testClient := &Client{
		httpClient: utils.NewHTTPClient(
			utils.WithBaseURL(server.URL),
		),
	}

	// Test the function
	txs, err := testClient.GetTransactions(context.Background(), "testAddress", Filters{})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify response
	if len(txs) != 1 {
		t.Fatalf("Expected 1 transaction, got %d", len(txs))
	}
	tx := txs[0]
	if tx.Signature != "testSignature" {
		t.Errorf("Expected signature testSignature, got %s", tx.Signature)
	}
	if tx.Description != "Test Transaction" {
		t.Errorf("Expected description 'Test Transaction', got %s", tx.Description)
	}
	if tx.Type != "TRANSFER" {
		t.Errorf("Expected type TRANSFER, got %s", tx.Type)
	}
	if tx.Fee != 5000 {
		t.Errorf("Expected fee 5000, got %d", tx.Fee)
	}
	if tx.Slot != 12345678 {
		t.Errorf("Expected slot 12345678, got %d", tx.Slot)
	}
	if tx.Timestamp != 1667289600 {
		t.Errorf("Expected timestamp 1667289600, got %d", tx.Timestamp)
	}
}
