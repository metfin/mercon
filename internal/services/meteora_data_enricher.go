package services

import (
	"fmt"
	"sync"
	"time"

	"github.com/wnt/mercon/internal/models"
	"gorm.io/gorm"
)

// MeteoraDataEnricher enriches Meteora data with USD values from the API
type MeteoraDataEnricher struct {
	db        *gorm.DB
	apiClient *MeteoraPubClient
	mutex     sync.Mutex
	// Cache price data to avoid excessive API calls
	pairPriceCache map[string]pairPriceData
}

type pairPriceData struct {
	price      float64
	lastUpdate time.Time
}

// NewMeteoraDataEnricher creates a new data enricher service
func NewMeteoraDataEnricher(db *gorm.DB) *MeteoraDataEnricher {
	return &MeteoraDataEnricher{
		db:             db,
		apiClient:      NewMeteoraPubClient(),
		pairPriceCache: make(map[string]pairPriceData),
	}
}

// EnrichPairs updates all pairs with USD data
func (e *MeteoraDataEnricher) EnrichPairs() error {
	// Get all pairs from the database
	var pairs []models.MeteoraPair
	if err := e.db.Find(&pairs).Error; err != nil {
		return fmt.Errorf("failed to fetch pairs: %w", err)
	}

	// Process each pair
	for _, pair := range pairs {
		if err := e.enrichPair(&pair); err != nil {
			fmt.Printf("Error enriching pair %s: %v\n", pair.Address, err)
			continue
		}
	}

	return nil
}

// enrichPair updates a single pair with data from the API
func (e *MeteoraDataEnricher) enrichPair(pair *models.MeteoraPair) error {
	// Check if we need to fetch new data
	now := time.Now()
	needsUpdate := pair.LastPriceUpdate.IsZero() || now.Sub(pair.LastPriceUpdate) > 1*time.Hour

	// If we need new data, fetch it from the API
	if needsUpdate {
		pairInfo, err := e.apiClient.GetPair(pair.Address)
		if err != nil {
			return fmt.Errorf("failed to fetch pair data from API: %w", err)
		}

		// Update pair with the new data
		pair.CurrentPrice = pairInfo.CurrentPrice
		pair.APR = pairInfo.Apr
		pair.APY = pairInfo.Apy
		pair.Fees24h = pairInfo.Fees24h
		pair.Volume24h = pairInfo.TradeVolume24h
		pair.LastPriceUpdate = now

		// Calculate USD values for reserves
		// Note: These calculations might need adjustment based on token decimals
		pair.ReserveXUSD = float64(pairInfo.ReserveXAmount) * pairInfo.CurrentPrice
		pair.ReserveYUSD = float64(pairInfo.ReserveYAmount)
		pair.TVL = pair.ReserveXUSD + pair.ReserveYUSD

		// Cache the price data
		e.mutex.Lock()
		e.pairPriceCache[pair.Address] = pairPriceData{
			price:      pairInfo.CurrentPrice,
			lastUpdate: now,
		}
		e.mutex.Unlock()

		// Save the updated pair
		if err := e.db.Save(pair).Error; err != nil {
			return fmt.Errorf("failed to save pair: %w", err)
		}
	}

	return nil
}

// EnrichPositions updates all positions with USD data
func (e *MeteoraDataEnricher) EnrichPositions() error {
	// Get all active positions from the database
	var positions []models.MeteoraPosition
	if err := e.db.Where("status = ?", "active").Find(&positions).Error; err != nil {
		return fmt.Errorf("failed to fetch positions: %w", err)
	}

	// Process each position
	for _, position := range positions {
		if err := e.enrichPosition(&position); err != nil {
			fmt.Printf("Error enriching position %s: %v\n", position.Address, err)
			continue
		}
	}

	return nil
}

// enrichPosition updates a single position with data from the API
func (e *MeteoraDataEnricher) enrichPosition(position *models.MeteoraPosition) error {
	// Check if we need to fetch new data
	now := time.Now()
	needsUpdate := position.LastDataUpdate.IsZero() || now.Sub(position.LastDataUpdate) > 6*time.Hour

	// If we need new data, fetch it from the API
	if needsUpdate {
		posInfo, err := e.apiClient.GetPosition(position.Address)
		if err != nil {
			return fmt.Errorf("failed to fetch position data from API: %w", err)
		}

		// Update position with the new data
		position.FeeAPY24h = posInfo.FeeApy24h
		position.FeeAPR24h = posInfo.FeeApr24h
		position.DailyFeeYield = posInfo.DailyFeeYield
		position.TotalFeeUSDClaimed = posInfo.TotalFeeUSDClaimed
		position.LastDataUpdate = now

		// Save the updated position
		if err := e.db.Save(position).Error; err != nil {
			return fmt.Errorf("failed to save position: %w", err)
		}
	}

	return nil
}

// EnrichSwap adds USD values to a swap
func (e *MeteoraDataEnricher) EnrichSwap(swap *models.MeteoraSwap) error {
	// First get pair to know the price
	var pair models.MeteoraPair
	if err := e.db.First(&pair, swap.PairID).Error; err != nil {
		return fmt.Errorf("failed to fetch pair for swap: %w", err)
	}

	// Ensure pair has current price data
	if err := e.enrichPair(&pair); err != nil {
		return fmt.Errorf("failed to enrich pair for swap: %w", err)
	}

	// Calculate USD values
	price := pair.CurrentPrice
	swap.TokenPrice = price

	// Determine which token is being swapped in
	if swap.SwapForY {
		// X -> Y swap
		swap.AmountInUSD = float64(swap.AmountIn) * price
		swap.AmountOutUSD = float64(swap.AmountOut)
	} else {
		// Y -> X swap
		swap.AmountInUSD = float64(swap.AmountIn)
		swap.AmountOutUSD = float64(swap.AmountOut) * price
	}

	// Calculate fee in USD
	// Note: This is a simplified approach. Real fee calculation may be more complex.
	if swap.SwapForY {
		swap.FeeUSD = float64(swap.Fee) * price
	} else {
		swap.FeeUSD = float64(swap.Fee)
	}

	// Save the updated swap
	if err := e.db.Save(swap).Error; err != nil {
		return fmt.Errorf("failed to save swap: %w", err)
	}

	return nil
}

// EnrichFeeClaim adds USD values to a fee claim
func (e *MeteoraDataEnricher) EnrichFeeClaim(claim *models.MeteoraFeeClaim) error {
	// First get position to establish pair relationship
	var position models.MeteoraPosition
	if err := e.db.First(&position, claim.PositionID).Error; err != nil {
		return fmt.Errorf("failed to fetch position for fee claim: %w", err)
	}

	// Then get pair to know the price
	var pair models.MeteoraPair
	if err := e.db.First(&pair, position.PairID).Error; err != nil {
		return fmt.Errorf("failed to fetch pair for fee claim: %w", err)
	}

	// Ensure pair has current price data
	if err := e.enrichPair(&pair); err != nil {
		return fmt.Errorf("failed to enrich pair for fee claim: %w", err)
	}

	// Try to get more accurate data from API
	positionClaims, err := e.apiClient.GetClaimFees(position.Address)
	if err == nil && len(positionClaims) > 0 {
		// Find the matching claim by txID
		var tx models.Transaction
		if err := e.db.First(&tx, claim.TransactionID).Error; err == nil {
			for _, apiClaim := range positionClaims {
				if apiClaim.TxID == tx.Signature {
					claim.AmountXUSD = apiClaim.TokenXUSDAmount
					claim.AmountYUSD = apiClaim.TokenYUSDAmount
					claim.TotalValueUSD = apiClaim.TokenXUSDAmount + apiClaim.TokenYUSDAmount
					claim.TokenPrice = pair.CurrentPrice

					// Save the updated claim
					if err := e.db.Save(claim).Error; err != nil {
						return fmt.Errorf("failed to save fee claim: %w", err)
					}

					return nil
				}
			}
		}
	}

	// If we couldn't find match in API data, calculate ourselves
	claim.TokenPrice = pair.CurrentPrice
	claim.AmountXUSD = float64(claim.AmountX) * pair.CurrentPrice
	claim.AmountYUSD = float64(claim.AmountY)
	claim.TotalValueUSD = claim.AmountXUSD + claim.AmountYUSD

	// Save the updated claim
	if err := e.db.Save(claim).Error; err != nil {
		return fmt.Errorf("failed to save fee claim: %w", err)
	}

	return nil
}

// EnrichLiquidityAddition adds USD values to a liquidity addition
func (e *MeteoraDataEnricher) EnrichLiquidityAddition(addition *models.MeteoraLiquidityAddition) error {
	// Get pair to know the price
	var pair models.MeteoraPair
	if err := e.db.First(&pair, addition.PairID).Error; err != nil {
		return fmt.Errorf("failed to fetch pair for liquidity addition: %w", err)
	}

	// Ensure pair has current price data
	if err := e.enrichPair(&pair); err != nil {
		return fmt.Errorf("failed to enrich pair for liquidity addition: %w", err)
	}

	// Get position to get more accurate data
	var position models.MeteoraPosition
	if err := e.db.First(&position, addition.PositionID).Error; err != nil {
		return fmt.Errorf("failed to fetch position for liquidity addition: %w", err)
	}

	// Try to get more accurate data from API
	deposits, err := e.apiClient.GetDeposits(position.Address)
	if err == nil && len(deposits) > 0 {
		// Find the matching deposit by txID
		var tx models.Transaction
		if err := e.db.First(&tx, addition.TransactionID).Error; err == nil {
			for _, deposit := range deposits {
				if deposit.TxID == tx.Signature {
					addition.AmountXUSD = deposit.TokenXUSDAmount
					addition.AmountYUSD = deposit.TokenYUSDAmount
					addition.TotalValueUSD = deposit.TokenXUSDAmount + deposit.TokenYUSDAmount
					addition.TokenPrice = deposit.Price

					// Save the updated addition
					if err := e.db.Save(addition).Error; err != nil {
						return fmt.Errorf("failed to save liquidity addition: %w", err)
					}

					return nil
				}
			}
		}
	}

	// If we couldn't find match in API data, calculate ourselves
	addition.TokenPrice = pair.CurrentPrice
	addition.AmountXUSD = float64(addition.AmountX) * pair.CurrentPrice
	addition.AmountYUSD = float64(addition.AmountY)
	addition.TotalValueUSD = addition.AmountXUSD + addition.AmountYUSD

	// Save the updated addition
	if err := e.db.Save(addition).Error; err != nil {
		return fmt.Errorf("failed to save liquidity addition: %w", err)
	}

	return nil
}

// EnrichLiquidityRemoval adds USD values to a liquidity removal
func (e *MeteoraDataEnricher) EnrichLiquidityRemoval(removal *models.MeteoraLiquidityRemoval) error {
	// Get pair to know the price
	var pair models.MeteoraPair
	if err := e.db.First(&pair, removal.PairID).Error; err != nil {
		return fmt.Errorf("failed to fetch pair for liquidity removal: %w", err)
	}

	// Ensure pair has current price data
	if err := e.enrichPair(&pair); err != nil {
		return fmt.Errorf("failed to enrich pair for liquidity removal: %w", err)
	}

	// Get position to get more accurate data
	var position models.MeteoraPosition
	if err := e.db.First(&position, removal.PositionID).Error; err != nil {
		return fmt.Errorf("failed to fetch position for liquidity removal: %w", err)
	}

	// Try to get more accurate data from API
	withdraws, err := e.apiClient.GetWithdraws(position.Address)
	if err == nil && len(withdraws) > 0 {
		// Find the matching withdrawal by txID
		var tx models.Transaction
		if err := e.db.First(&tx, removal.TransactionID).Error; err == nil {
			for _, withdraw := range withdraws {
				if withdraw.TxID == tx.Signature {
					removal.AmountXRemoved = uint64(withdraw.TokenXAmount)
					removal.AmountYRemoved = uint64(withdraw.TokenYAmount)
					removal.AmountXRemovedUSD = withdraw.TokenXUSDAmount
					removal.AmountYRemovedUSD = withdraw.TokenYUSDAmount
					removal.TotalValueUSD = withdraw.TokenXUSDAmount + withdraw.TokenYUSDAmount
					removal.TokenPrice = withdraw.Price

					// Save the updated removal
					if err := e.db.Save(removal).Error; err != nil {
						return fmt.Errorf("failed to save liquidity removal: %w", err)
					}

					return nil
				}
			}
		}
	}

	// If we couldn't find match in API data and no amounts are set, we can't calculate
	if removal.AmountXRemoved == 0 && removal.AmountYRemoved == 0 {
		return nil
	}

	// Calculate with what we have
	removal.TokenPrice = pair.CurrentPrice
	removal.AmountXRemovedUSD = float64(removal.AmountXRemoved) * pair.CurrentPrice
	removal.AmountYRemovedUSD = float64(removal.AmountYRemoved)
	removal.TotalValueUSD = removal.AmountXRemovedUSD + removal.AmountYRemovedUSD

	// Save the updated removal
	if err := e.db.Save(removal).Error; err != nil {
		return fmt.Errorf("failed to save liquidity removal: %w", err)
	}

	return nil
}

// PostProcessTransaction enriches all Meteora entities related to a transaction with USD values
func (e *MeteoraDataEnricher) PostProcessTransaction(tx *models.Transaction) error {
	// Enrich all swaps
	var swaps []models.MeteoraSwap
	if err := e.db.Where("transaction_id = ?", tx.ID).Find(&swaps).Error; err != nil {
		return fmt.Errorf("failed to fetch swaps: %w", err)
	}
	for i := range swaps {
		if err := e.EnrichSwap(&swaps[i]); err != nil {
			fmt.Printf("Error enriching swap: %v\n", err)
		}
	}

	// Enrich all liquidity additions
	var additions []models.MeteoraLiquidityAddition
	if err := e.db.Where("transaction_id = ?", tx.ID).Find(&additions).Error; err != nil {
		return fmt.Errorf("failed to fetch additions: %w", err)
	}
	for i := range additions {
		if err := e.EnrichLiquidityAddition(&additions[i]); err != nil {
			fmt.Printf("Error enriching liquidity addition: %v\n", err)
		}
	}

	// Enrich all liquidity removals
	var removals []models.MeteoraLiquidityRemoval
	if err := e.db.Where("transaction_id = ?", tx.ID).Find(&removals).Error; err != nil {
		return fmt.Errorf("failed to fetch removals: %w", err)
	}
	for i := range removals {
		if err := e.EnrichLiquidityRemoval(&removals[i]); err != nil {
			fmt.Printf("Error enriching liquidity removal: %v\n", err)
		}
	}

	// Enrich all fee claims
	var feeClaims []models.MeteoraFeeClaim
	if err := e.db.Where("transaction_id = ?", tx.ID).Find(&feeClaims).Error; err != nil {
		return fmt.Errorf("failed to fetch fee claims: %w", err)
	}
	for i := range feeClaims {
		if err := e.EnrichFeeClaim(&feeClaims[i]); err != nil {
			fmt.Printf("Error enriching fee claim: %v\n", err)
		}
	}

	return nil
}

// ScheduleRegularUpdates runs periodic updates of USD values
func (e *MeteoraDataEnricher) ScheduleRegularUpdates(interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		for {
			<-ticker.C
			fmt.Println("Running scheduled USD data update...")

			if err := e.EnrichPairs(); err != nil {
				fmt.Printf("Error updating pairs: %v\n", err)
			}

			if err := e.EnrichPositions(); err != nil {
				fmt.Printf("Error updating positions: %v\n", err)
			}
		}
	}()
}
