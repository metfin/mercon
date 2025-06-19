package worker

import (
	"context"
	"fmt"
	"time"

	"github.com/metfin/core/rawchain"
	"github.com/rs/zerolog"
	"github.com/wnt/mercon/internal/logger"
	"github.com/wnt/mercon/internal/metrics"
	"github.com/wnt/mercon/internal/queue"
	"github.com/wnt/mercon/internal/rpc"
)

// Worker represents a single wallet processing worker
type Worker struct {
	id       string
	queue    *queue.Client
	fetcher  *rpc.Fetcher
	logger   zerolog.Logger
	stopped  bool
}

// NewWorker creates a new worker instance
func NewWorker(id string, queueClient *queue.Client, fetcher *rpc.Fetcher, baseLogger zerolog.Logger) *Worker {
	return &Worker{
		id:      id,
		queue:   queueClient,
		fetcher: fetcher,
		logger:  logger.WithWorker(baseLogger, id),
	}
}

// Start begins the worker processing loop
func (w *Worker) Start(ctx context.Context) error {
	w.logger.Info().Msg("Starting worker")
	
	for {
		select {
		case <-ctx.Done():
			w.logger.Info().Msg("Worker received shutdown signal")
			return ctx.Err()
		default:
			if w.stopped {
				w.logger.Info().Msg("Worker stopped")
				return nil
			}
			
			// Process a single wallet
			if err := w.processWallet(ctx); err != nil {
				w.logger.Error().Err(err).Msg("Failed to process wallet")
				// Continue processing other wallets even if one fails
				
				// Brief pause to avoid tight error loops
				select {
				case <-time.After(5 * time.Second):
				case <-ctx.Done():
					return ctx.Err()
				}
			}
		}
	}
}

// Stop signals the worker to stop gracefully
func (w *Worker) Stop() {
	w.stopped = true
	w.logger.Info().Msg("Worker stop signal received")
}

// processWallet handles the complete lifecycle of processing a single wallet
func (w *Worker) processWallet(ctx context.Context) error {
	// Fetch a wallet from the queue
	wallet, err := w.queue.PopWallet(ctx)
	if err != nil {
		return fmt.Errorf("failed to pop wallet from queue: %w", err)
	}
	
	// No wallet available
	if wallet == "" {
		// Brief pause when queue is empty to avoid spinning
		select {
		case <-time.After(10 * time.Second):
		case <-ctx.Done():
			return ctx.Err()
		}
		return nil
	}
	
	// Mark wallet as in-flight
	if err := w.queue.SetInFlight(ctx, wallet, w.id); err != nil {
		w.logger.Error().Err(err).Str("wallet", wallet).Msg("Failed to mark wallet as in-flight")
		// Re-queue the wallet since we couldn't track it
		if requeueErr := w.queue.PushWallet(ctx, wallet, 0); requeueErr != nil {
			w.logger.Error().Err(requeueErr).Str("wallet", wallet).Msg("Failed to requeue wallet after in-flight error")
		}
		return err
	}
	
	walletLogger := logger.WithWallet(w.logger, wallet)
	startTime := time.Now()
	
	walletLogger.Info().Msg("Starting wallet processing")
	
	// Process the wallet
	err = w.scrapeWallet(ctx, wallet, walletLogger)
	duration := time.Since(startTime)
	
	// Record metrics
	metrics.RecordWalletScrape(duration.Seconds())
	metrics.RecordWorkerTaskDuration("wallet_scrape", w.id, duration.Seconds())
	
	// Remove from in-flight tracking
	if removeErr := w.queue.RemoveInFlight(ctx, wallet); removeErr != nil {
		walletLogger.Error().Err(removeErr).Msg("Failed to remove wallet from in-flight tracking")
	}
	
	if err != nil {
		walletLogger.Error().Err(err).Dur("duration", duration).Msg("Failed to process wallet")
		
		// Re-queue with lower priority (higher score) on failure
		if requeueErr := w.queue.PushWallet(ctx, wallet, float64(time.Now().Unix())); requeueErr != nil {
			walletLogger.Error().Err(requeueErr).Msg("Failed to requeue failed wallet")
		}
		
		return fmt.Errorf("wallet processing failed: %w", err)
	}
	
	walletLogger.Info().Dur("duration", duration).Msg("Wallet processing completed successfully")
	return nil
}

// scrapeWallet handles the actual wallet scraping process
func (w *Worker) scrapeWallet(ctx context.Context, wallet string, logger zerolog.Logger) error {
	// Get last processed signature for resume capability
	lastSig, err := w.queue.GetProgress(ctx, wallet)
	if err != nil {
		return fmt.Errorf("failed to get wallet progress: %w", err)
	}
	
	if lastSig != "" {
		logger.Debug().Str("last_signature", lastSig).Msg("Resuming from last processed signature")
	} else {
		logger.Debug().Msg("Starting fresh wallet scrape")
	}
	
	// Fetch signatures in batches
	const batchSize = 1000
	processedCount := 0
	before := ""
	
	// If we have a last signature, start from there
	if lastSig != "" {
		before = lastSig
	}
	
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		
		// Fetch batch of signatures
		signatures, err := w.fetcher.FetchSignatures(ctx, wallet, before, batchSize)
		if err != nil {
			return fmt.Errorf("failed to fetch signatures: %w", err)
		}
		
		// No more signatures
		if len(signatures) == 0 {
			break
		}
		
		logger.Debug().Int("signatures", len(signatures)).Msg("Fetched signature batch")
		
		// Process each signature in the batch
		for _, signature := range signatures {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}
			
			// Skip if this is the last processed signature (avoid duplicates)
			if signature == lastSig {
				continue
			}
			
			if err := w.processTransaction(ctx, signature, logger); err != nil {
				logger.Warn().Err(err).Str("signature", signature).Msg("Failed to process transaction, continuing")
				metrics.RecordTransactionProcessed("failed")
				continue
			}
			
			metrics.RecordTransactionProcessed("success")
			processedCount++
			
			// Update progress every 100 transactions
			if processedCount%100 == 0 {
				if err := w.queue.SetProgress(ctx, wallet, signature); err != nil {
					logger.Warn().Err(err).Str("signature", signature).Msg("Failed to update progress")
				} else {
					logger.Debug().Int("processed", processedCount).Str("signature", signature).Msg("Progress updated")
				}
			}
		}
		
		// Prepare for next batch
		if len(signatures) > 0 {
			before = signatures[len(signatures)-1]
			// Update final progress
			if err := w.queue.SetProgress(ctx, wallet, before); err != nil {
				logger.Warn().Err(err).Str("signature", before).Msg("Failed to update final progress")
			}
		}
		
		// If we got fewer signatures than batch size, we're done
		if len(signatures) < batchSize {
			break
		}
		
		// Brief pause between batches to be nice to RPC endpoints
		select {
		case <-time.After(100 * time.Millisecond):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	
	logger.Info().Int("total_processed", processedCount).Msg("Wallet scraping completed")
	return nil
}

// processTransaction fetches and processes a single transaction
func (w *Worker) processTransaction(ctx context.Context, signature string, logger zerolog.Logger) error {
	// Fetch the transaction details
	rpcTx, err := w.fetcher.FetchTransaction(ctx, signature)
	if err != nil {
		return fmt.Errorf("failed to fetch transaction %s: %w", signature, err)
	}
	
	if rpcTx == nil {
		return fmt.Errorf("transaction %s not found", signature)
	}
	
	// Convert RPC transaction to raw chain format
	rawTx := map[string]interface{}{
		"slot":        rpcTx.Slot,
		"blockTime":   rpcTx.BlockTime,
		"transaction": rpcTx.Transaction,
		"meta":        rpcTx.Meta,
	}
	
	// Parse RPC transaction for insertion
	chainTx, err := parseRPCTransactionForInsertion(rawTx)
	if err != nil {
		return fmt.Errorf("failed to parse transaction %s: %w", signature, err)
	}
	
	// Insert into raw chain database
	if err := insertTransactionToRawChain(ctx, chainTx); err != nil {
		return fmt.Errorf("failed to insert transaction %s: %w", signature, err)
	}
	
	logger.Debug().
		Str("signature", signature).
		Uint64("slot", rpcTx.Slot).
		Interface("block_time", rpcTx.BlockTime).
		Int("instructions", len(chainTx.Instructions)).
		Int("token_balances", len(chainTx.TokenBalances)).
		Msg("Transaction processed and inserted successfully")
	
	return nil
}

// parseRPCTransactionForInsertion converts RPC transaction data to rawchain format
func parseRPCTransactionForInsertion(rpcTx map[string]interface{}) (*rawchain.Transaction, error) {
	return rawchain.ParseRPCTransaction(rpcTx)
}

// insertTransactionToRawChain inserts a transaction using the rawchain package
func insertTransactionToRawChain(ctx context.Context, tx *rawchain.Transaction) error {
	if err := rawchain.InsertTransaction(ctx, tx); err != nil {
		metrics.RecordDatabaseOperation("insert", "failed")
		return err
	}
	
	metrics.RecordDatabaseOperation("insert", "success")
	return nil
} 