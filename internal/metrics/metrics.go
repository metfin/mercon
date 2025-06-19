package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// WalletQueueLength tracks the number of wallets in the queue
	WalletQueueLength = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "mercon_wallet_queue_length",
		Help: "The number of wallets currently in the queue",
	})

	// WorkersActive tracks the number of active workers
	WorkersActive = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "mercon_workers_active",
		Help: "The number of workers currently active",
	})

	// RPCRequestsTotal tracks RPC requests by status
	RPCRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "mercon_rpc_requests_total",
			Help: "The total number of RPC requests",
		},
		[]string{"status"},
	)

	// WalletScrapeSeconds tracks time taken to scrape wallets
	WalletScrapeSeconds = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "mercon_wallet_scrape_seconds",
		Help:    "Time taken to scrape a wallet in seconds",
		Buckets: prometheus.ExponentialBuckets(1, 2, 10), // 1s to ~17min
	})

	// TransactionsProcessed tracks the number of transactions processed
	TransactionsProcessed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "mercon_transactions_processed_total",
			Help: "The total number of transactions processed",
		},
		[]string{"status"}, // success, failed
	)

	// DatabaseOperations tracks database operations
	DatabaseOperations = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "mercon_database_operations_total",
			Help: "The total number of database operations",
		},
		[]string{"operation", "status"}, // insert/update, success/failed
	)

	// RPCEndpointHealth tracks RPC endpoint health
	RPCEndpointHealth = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "mercon_rpc_endpoint_health",
			Help: "Health status of RPC endpoints (1 = healthy, 0 = unhealthy)",
		},
		[]string{"endpoint"},
	)

	// WorkerTaskDuration tracks how long workers spend on tasks
	WorkerTaskDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "mercon_worker_task_duration_seconds",
			Help:    "Time taken by workers to complete tasks",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"task_type", "worker_id"},
	)
)

// RecordRPCRequest records an RPC request with the given status
func RecordRPCRequest(status string) {
	RPCRequestsTotal.WithLabelValues(status).Inc()
}

// RecordWalletScrape records the time taken to scrape a wallet
func RecordWalletScrape(duration float64) {
	WalletScrapeSeconds.Observe(duration)
}

// RecordTransactionProcessed records a processed transaction
func RecordTransactionProcessed(status string) {
	TransactionsProcessed.WithLabelValues(status).Inc()
}

// RecordDatabaseOperation records a database operation
func RecordDatabaseOperation(operation, status string) {
	DatabaseOperations.WithLabelValues(operation, status).Inc()
}

// SetRPCEndpointHealth sets the health status of an RPC endpoint
func SetRPCEndpointHealth(endpoint string, healthy bool) {
	value := 0.0
	if healthy {
		value = 1.0
	}
	RPCEndpointHealth.WithLabelValues(endpoint).Set(value)
}

// RecordWorkerTaskDuration records the time taken by a worker to complete a task
func RecordWorkerTaskDuration(taskType, workerID string, duration float64) {
	WorkerTaskDuration.WithLabelValues(taskType, workerID).Observe(duration)
} 