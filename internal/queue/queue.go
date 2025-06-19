package queue

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
)

// Client wraps Redis operations for Mercon queue management
type Client struct {
	client *redis.Client
	logger zerolog.Logger
}

// NewClient creates a new Redis queue client
func NewClient(redisURL string, logger zerolog.Logger) (*Client, error) {
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Redis URL: %w", err)
	}

	client := redis.NewClient(opt)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	logger.Info().Str("redis_url", redisURL).Msg("Connected to Redis successfully")

	return &Client{
		client: client,
		logger: logger.With().Str("component", "queue").Logger(),
	}, nil
}

// PopWallet removes and returns the wallet with the lowest score (highest priority)
func (c *Client) PopWallet(ctx context.Context) (string, error) {
	result, err := c.client.ZPopMin(ctx, "wallet_queue", 1).Result()
	if err != nil {
		if err == redis.Nil {
			return "", nil // No wallets in queue
		}
		return "", fmt.Errorf("failed to pop wallet from queue: %w", err)
	}

	if len(result) == 0 {
		return "", nil // No wallets in queue
	}

	wallet := result[0].Member.(string)
	c.logger.Debug().Str("wallet", wallet).Msg("Popped wallet from queue")
	return wallet, nil
}

// PushWallet adds a wallet to the queue with the specified priority
func (c *Client) PushWallet(ctx context.Context, addr string, priority float64) error {
	err := c.client.ZAdd(ctx, "wallet_queue", redis.Z{
		Score:  priority,
		Member: addr,
	}).Err()

	if err != nil {
		return fmt.Errorf("failed to push wallet to queue: %w", err)
	}

	c.logger.Debug().
		Str("wallet", addr).
		Float64("priority", priority).
		Msg("Pushed wallet to queue")
	
	return nil
}

// SetInFlight marks a wallet as being processed by a worker
func (c *Client) SetInFlight(ctx context.Context, addr, worker string) error {
	value := fmt.Sprintf("%s,%d", worker, time.Now().Unix())
	err := c.client.HSet(ctx, "wallet_inflight", addr, value).Err()
	
	if err != nil {
		return fmt.Errorf("failed to set wallet in-flight: %w", err)
	}

	c.logger.Debug().
		Str("wallet", addr).
		Str("worker", worker).
		Msg("Marked wallet as in-flight")
	
	return nil
}

// RemoveInFlight removes a wallet from the in-flight tracking
func (c *Client) RemoveInFlight(ctx context.Context, addr string) error {
	err := c.client.HDel(ctx, "wallet_inflight", addr).Err()
	
	if err != nil {
		return fmt.Errorf("failed to remove wallet from in-flight: %w", err)
	}

	c.logger.Debug().Str("wallet", addr).Msg("Removed wallet from in-flight")
	return nil
}

// GetProgress retrieves the last processed signature for a wallet
func (c *Client) GetProgress(ctx context.Context, addr string) (string, error) {
	result, err := c.client.HGet(ctx, "wallet_progress", addr).Result()
	if err != nil {
		if err == redis.Nil {
			return "", nil // No progress recorded
		}
		return "", fmt.Errorf("failed to get wallet progress: %w", err)
	}

	return result, nil
}

// SetProgress updates the last processed signature for a wallet
func (c *Client) SetProgress(ctx context.Context, addr, sig string) error {
	err := c.client.HSet(ctx, "wallet_progress", addr, sig).Err()
	
	if err != nil {
		return fmt.Errorf("failed to set wallet progress: %w", err)
	}

	c.logger.Debug().
		Str("wallet", addr).
		Str("signature", sig).
		Msg("Updated wallet progress")
	
	return nil
}

// GetQueueLength returns the number of wallets in the queue
func (c *Client) GetQueueLength(ctx context.Context) (int64, error) {
	length, err := c.client.ZCard(ctx, "wallet_queue").Result()
	if err != nil {
		return 0, fmt.Errorf("failed to get queue length: %w", err)
	}
	return length, nil
}

// GetInFlightWallets returns all wallets currently being processed
func (c *Client) GetInFlightWallets(ctx context.Context) (map[string]string, error) {
	result, err := c.client.HGetAll(ctx, "wallet_inflight").Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get in-flight wallets: %w", err)
	}
	return result, nil
}

// RequeueStuckWallets moves wallets that have been in-flight too long back to the queue
func (c *Client) RequeueStuckWallets(ctx context.Context, timeoutMinutes int) error {
	inFlight, err := c.GetInFlightWallets(ctx)
	if err != nil {
		return fmt.Errorf("failed to get in-flight wallets: %w", err)
	}

	cutoff := time.Now().Add(-time.Duration(timeoutMinutes) * time.Minute).Unix()
	requeuedCount := 0

	for wallet, value := range inFlight {
		parts := splitValue(value)
		if len(parts) != 2 {
			c.logger.Warn().Str("wallet", wallet).Str("value", value).Msg("Invalid in-flight value format")
			continue
		}

		startTime, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			c.logger.Warn().Str("wallet", wallet).Str("value", value).Msg("Invalid timestamp in in-flight value")
			continue
		}

		if startTime < cutoff {
			// Wallet has been stuck too long, requeue it
			if err := c.PushWallet(ctx, wallet, 0); err != nil {
				c.logger.Error().Err(err).Str("wallet", wallet).Msg("Failed to requeue stuck wallet")
				continue
			}

			if err := c.RemoveInFlight(ctx, wallet); err != nil {
				c.logger.Error().Err(err).Str("wallet", wallet).Msg("Failed to remove requeued wallet from in-flight")
			}

			requeuedCount++
			c.logger.Info().
				Str("wallet", wallet).
				Str("worker", parts[0]).
				Int64("stuck_minutes", (time.Now().Unix()-startTime)/60).
				Msg("Requeued stuck wallet")
		}
	}

	if requeuedCount > 0 {
		c.logger.Info().Int("count", requeuedCount).Msg("Requeued stuck wallets")
	}

	return nil
}

// Close closes the Redis connection
func (c *Client) Close() error {
	return c.client.Close()
}

// splitValue splits the in-flight value format "worker,timestamp"
func splitValue(value string) []string {
	parts := make([]string, 0, 2)
	commaIndex := -1
	
	for i, char := range value {
		if char == ',' {
			commaIndex = i
			break
		}
	}
	
	if commaIndex == -1 {
		return []string{value}
	}
	
	parts = append(parts, value[:commaIndex])
	parts = append(parts, value[commaIndex+1:])
	return parts
} 