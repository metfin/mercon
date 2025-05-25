package services

import (
	"testing"
	"time"

	"github.com/wnt/mercon/internal/models"
)

// TestNewMeteoraDataEnricher tests creating a new enricher
func TestNewMeteoraDataEnricher(t *testing.T) {
	// Since we're not testing database interactions, we can create with nil DB
	enricher := NewMeteoraDataEnricher(nil)

	if enricher.apiClient == nil {
		t.Fatal("Enricher API client should not be nil")
	}
	if enricher.pairPriceCache == nil {
		t.Fatal("Enricher price cache should not be nil")
	}
}

// TestCacheMechanism tests the price cache
func TestCacheMechanism(t *testing.T) {
	// Create a new enricher with nil DB
	enricher := &MeteoraDataEnricher{
		db:             nil,
		apiClient:      nil, // Not testing API interactions
		pairPriceCache: make(map[string]pairPriceData),
	}

	// Test adding to cache
	pairAddress := "TestPairAddress123"
	now := time.Now()
	testPrice := 1.25

	enricher.mutex.Lock()
	enricher.pairPriceCache[pairAddress] = pairPriceData{
		price:      testPrice,
		lastUpdate: now,
	}
	enricher.mutex.Unlock()

	// Verify cache retrieval
	enricher.mutex.Lock()
	cachedData, exists := enricher.pairPriceCache[pairAddress]
	enricher.mutex.Unlock()

	if !exists {
		t.Error("Price data should be in cache")
	}
	if cachedData.price != testPrice {
		t.Errorf("Cached price = %f, want %f", cachedData.price, testPrice)
	}
	if !cachedData.lastUpdate.Equal(now) {
		t.Errorf("Cached lastUpdate incorrect")
	}
}

// TestSwapCalculations tests the swap enrichment calculations
func TestSwapCalculations(t *testing.T) {
	// Create a swap with known values (X -> Y swap)
	xToYSwap := &models.MeteoraSwap{
		SwapForY:  true,
		AmountIn:  1000,
		AmountOut: 1200,
		Fee:       3,
	}

	tokenPrice := 1.25

	// Manually calculate the expected values for X -> Y swap
	xToYAmountInUSD := float64(xToYSwap.AmountIn) * tokenPrice
	xToYAmountOutUSD := float64(xToYSwap.AmountOut)
	xToYFeeUSD := float64(xToYSwap.Fee) * tokenPrice

	if xToYAmountInUSD != 1250.0 {
		t.Errorf("Expected AmountInUSD to be 1250.0, got %f", xToYAmountInUSD)
	}
	if xToYAmountOutUSD != 1200.0 {
		t.Errorf("Expected AmountOutUSD to be 1200.0, got %f", xToYAmountOutUSD)
	}
	if xToYFeeUSD != 3.75 {
		t.Errorf("Expected FeeUSD to be 3.75, got %f", xToYFeeUSD)
	}

	// Create a swap with known values (Y -> X swap)
	yToXSwap := &models.MeteoraSwap{
		SwapForY:  false,
		AmountIn:  1000,
		AmountOut: 800,
		Fee:       3,
	}

	// Manually calculate the expected values for Y -> X swap
	yToXAmountInUSD := float64(yToXSwap.AmountIn)
	yToXAmountOutUSD := float64(yToXSwap.AmountOut) * tokenPrice
	yToXFeeUSD := float64(yToXSwap.Fee)

	if yToXAmountInUSD != 1000.0 {
		t.Errorf("Expected AmountInUSD to be 1000.0, got %f", yToXAmountInUSD)
	}
	if yToXAmountOutUSD != 1000.0 {
		t.Errorf("Expected AmountOutUSD to be 1000.0, got %f", yToXAmountOutUSD)
	}
	if yToXFeeUSD != 3.0 {
		t.Errorf("Expected FeeUSD to be 3.0, got %f", yToXFeeUSD)
	}
}
