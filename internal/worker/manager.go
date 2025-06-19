package worker

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/wnt/mercon/internal/config"
	"github.com/wnt/mercon/internal/metrics"
	"github.com/wnt/mercon/internal/queue"
	"github.com/wnt/mercon/internal/rpc"
	"golang.org/x/sync/errgroup"
)

// Manager manages a dynamic pool of workers
type Manager struct {
	config    config.Config
	queue     *queue.Client
	rpcPool   *rpc.Pool
	fetcher   *rpc.Fetcher
	workers   []*Worker
	logger    zerolog.Logger
	mutex     sync.RWMutex
	ctx       context.Context
	cancel    context.CancelFunc
	eg        *errgroup.Group
	stopped   bool
}

// NewManager creates a new worker manager
func NewManager(cfg config.Config, queueClient *queue.Client, rpcPool *rpc.Pool, logger zerolog.Logger) *Manager {
	ctx, cancel := context.WithCancel(context.Background())
	eg, egCtx := errgroup.WithContext(ctx)
	
	fetcher := rpc.NewFetcher(rpcPool, logger)
	
	manager := &Manager{
		config:  cfg,
		queue:   queueClient,
		rpcPool: rpcPool,
		fetcher: fetcher,
		workers: make([]*Worker, 0),
		logger:  logger.With().Str("component", "worker_manager").Logger(),
		ctx:     egCtx,
		cancel:  cancel,
		eg:      eg,
	}
	
	return manager
}

// Start begins the worker manager lifecycle
func (m *Manager) Start() error {
	m.logger.Info().
		Int("min_workers", m.config.MinWorkers).
		Int("max_workers", m.config.MaxWorkers).
		Msg("Starting worker manager")
	
	// Start initial workers
	if err := m.adjustWorkerCount(); err != nil {
		return fmt.Errorf("failed to start initial workers: %w", err)
	}
	
	// Start the scaling ticker
	m.eg.Go(func() error {
		return m.runScalingLoop()
	})
	
	// Start stuck wallet recovery
	m.eg.Go(func() error {
		return m.runStuckWalletRecovery()
	})
	
	// Start queue monitoring
	m.eg.Go(func() error {
		return m.runQueueMonitoring()
	})
	
	m.logger.Info().Msg("Worker manager started successfully")
	return nil
}

// Stop gracefully shuts down the worker manager
func (m *Manager) Stop() error {
	m.mutex.Lock()
	if m.stopped {
		m.mutex.Unlock()
		return nil
	}
	m.stopped = true
	m.mutex.Unlock()
	
	m.logger.Info().Msg("Stopping worker manager...")
	
	// Cancel context to signal all workers to stop
	m.cancel()
	
	// Wait for all workers to finish with timeout
	done := make(chan error, 1)
	go func() {
		done <- m.eg.Wait()
	}()
	
	select {
	case err := <-done:
		if err != nil {
			m.logger.Error().Err(err).Msg("Error during worker shutdown")
		}
	case <-time.After(30 * time.Second):
		m.logger.Warn().Msg("Worker shutdown timed out")
	}
	
	// Clear workers
	m.mutex.Lock()
	m.workers = nil
	m.mutex.Unlock()
	
	metrics.WorkersActive.Set(0)
	m.logger.Info().Msg("Worker manager stopped")
	return nil
}

// runScalingLoop handles automatic worker scaling every 30 seconds
func (m *Manager) runScalingLoop() error {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-m.ctx.Done():
			return m.ctx.Err()
		case <-ticker.C:
			if err := m.adjustWorkerCount(); err != nil {
				m.logger.Error().Err(err).Msg("Failed to adjust worker count")
			}
		}
	}
}

// adjustWorkerCount scales workers based on queue length
func (m *Manager) adjustWorkerCount() error {
	queueLength, err := m.queue.GetQueueLength(m.ctx)
	if err != nil {
		return fmt.Errorf("failed to get queue length: %w", err)
	}
	
	// Update queue length metric
	metrics.WalletQueueLength.Set(float64(queueLength))
	
	// Calculate desired worker count
	desiredWorkers := m.calculateDesiredWorkers(int(queueLength))
	
	m.mutex.Lock()
	currentWorkers := len(m.workers)
	m.mutex.Unlock()
	
	if desiredWorkers == currentWorkers {
		return nil // No change needed
	}
	
	m.logger.Info().
		Int("current_workers", currentWorkers).
		Int("desired_workers", desiredWorkers).
		Int64("queue_length", queueLength).
		Msg("Adjusting worker count")
	
	if desiredWorkers > currentWorkers {
		return m.addWorkers(desiredWorkers - currentWorkers)
	} else {
		return m.removeWorkers(currentWorkers - desiredWorkers)
	}
}

// calculateDesiredWorkers determines optimal worker count based on queue length
func (m *Manager) calculateDesiredWorkers(queueLength int) int {
	// Simple scaling algorithm: 1 worker per 10 wallets in queue
	desired := queueLength / 10
	if desired < m.config.MinWorkers {
		desired = m.config.MinWorkers
	}
	if desired > m.config.MaxWorkers {
		desired = m.config.MaxWorkers
	}
	return desired
}

// addWorkers creates and starts new workers
func (m *Manager) addWorkers(count int) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	for i := 0; i < count; i++ {
		workerID := fmt.Sprintf("worker-%d", len(m.workers)+1)
		worker := NewWorker(workerID, m.queue, m.fetcher, m.logger)
		
		// Start the worker
		m.eg.Go(func() error {
			return worker.Start(m.ctx)
		})
		
		m.workers = append(m.workers, worker)
		
		m.logger.Debug().
			Str("worker_id", workerID).
			Int("total_workers", len(m.workers)).
			Msg("Added worker")
	}
	
	// Update metrics
	metrics.WorkersActive.Set(float64(len(m.workers)))
	
	m.logger.Info().
		Int("added", count).
		Int("total_workers", len(m.workers)).
		Msg("Workers added")
	
	return nil
}

// removeWorkers gracefully stops and removes workers
func (m *Manager) removeWorkers(count int) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	if count > len(m.workers) {
		count = len(m.workers)
	}
	
	// Signal workers to stop (they will finish current work)
	workersToRemove := m.workers[len(m.workers)-count:]
	for _, worker := range workersToRemove {
		worker.Stop()
	}
	
	// Remove from slice
	m.workers = m.workers[:len(m.workers)-count]
	
	// Update metrics
	metrics.WorkersActive.Set(float64(len(m.workers)))
	
	m.logger.Info().
		Int("removed", count).
		Int("remaining_workers", len(m.workers)).
		Msg("Workers removed")
	
	return nil
}

// runStuckWalletRecovery periodically checks for and requeues stuck wallets
func (m *Manager) runStuckWalletRecovery() error {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	
	for {
		select {
		case <-m.ctx.Done():
			return m.ctx.Err()
		case <-ticker.C:
			if err := m.queue.RequeueStuckWallets(m.ctx, 15); err != nil {
				m.logger.Error().Err(err).Msg("Failed to requeue stuck wallets")
			}
		}
	}
}

// runQueueMonitoring periodically logs queue statistics
func (m *Manager) runQueueMonitoring() error {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	
	for {
		select {
		case <-m.ctx.Done():
			return m.ctx.Err()
		case <-ticker.C:
			queueLength, err := m.queue.GetQueueLength(m.ctx)
			if err != nil {
				m.logger.Error().Err(err).Msg("Failed to get queue length for monitoring")
				continue
			}
			
			inFlight, err := m.queue.GetInFlightWallets(m.ctx)
			if err != nil {
				m.logger.Error().Err(err).Msg("Failed to get in-flight wallets for monitoring")
				continue
			}
			
			m.mutex.RLock()
			activeWorkers := len(m.workers)
			m.mutex.RUnlock()
			
			healthyEndpoints := m.rpcPool.GetHealthyEndpointCount()
			
			m.logger.Info().
				Int64("queue_length", queueLength).
				Int("in_flight_wallets", len(inFlight)).
				Int("active_workers", activeWorkers).
				Int("healthy_endpoints", healthyEndpoints).
				Msg("Queue monitoring stats")
		}
	}
}

// GetStats returns current manager statistics
func (m *Manager) GetStats() map[string]interface{} {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	queueLength, _ := m.queue.GetQueueLength(context.Background())
	inFlight, _ := m.queue.GetInFlightWallets(context.Background())
	
	return map[string]interface{}{
		"active_workers":     len(m.workers),
		"queue_length":       queueLength,
		"in_flight_wallets":  len(inFlight),
		"healthy_endpoints":  m.rpcPool.GetHealthyEndpointCount(),
		"min_workers":        m.config.MinWorkers,
		"max_workers":        m.config.MaxWorkers,
	}
} 