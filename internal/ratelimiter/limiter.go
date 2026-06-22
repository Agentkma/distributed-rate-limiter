package ratelimiter

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	windowRequestLimit = 3
	windowDurationSec  = 60 * time.Second
	requestTimeoutSec  = 2 * time.Second
	minuteWindowLayout = "200601021504"
)

// Store is the Redis operations required by the rate limiter.
type Store interface {
	Incr(ctx context.Context, key string) (int64, error)
	Expire(ctx context.Context, key string, expiration time.Duration) (bool, error)
}

// clientStore adapts *redis.Client to the Store interface.
type clientStore struct {
	client *redis.Client
}

func (s *clientStore) Incr(ctx context.Context, key string) (int64, error) {
	return s.client.Incr(ctx, key).Result()
}

func (s *clientStore) Expire(ctx context.Context, key string, expiration time.Duration) (bool, error) {
	return s.client.Expire(ctx, key, expiration).Result()
}

// NewStore wraps a *redis.Client as a Store.
func NewStore(client *redis.Client) Store {
	return &clientStore{client: client}
}

func Allow(store Store, clientAddress string) bool {
	ctx, cancel := newRequestContext()
	defer cancel()

	rateLimitKey := buildRateLimitKey(clientAddress)

	count, ok := incrementRequestCount(ctx, store, rateLimitKey)
	if !ok {
		// Fail-open: allow requests when Redis is unavailable.
		return true
	}

	if isFirstRequestForWindow(count) {
		if !setWindowExpiration(ctx, store, rateLimitKey) {
			// Fail-open: allow requests when Redis is unavailable.
			return true
		}
	}

	return isWithinLimit(count)
}

func newRequestContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), requestTimeoutSec)
}

func buildRateLimitKey(clientAddress string) string {
	return fmt.Sprintf("rate:%s:%s", clientAddress, currentMinuteWindow())
}

func incrementRequestCount(ctx context.Context, store Store, key string) (int64, bool) {
	count, err := store.Incr(ctx, key)
	if err != nil {
		logRedisError("INCR", key, err)
		return 0, false
	}

	return count, true
}

func isFirstRequestForWindow(count int64) bool {
	return count == 1
}

func setWindowExpiration(ctx context.Context, store Store, key string) bool {
	if _, err := store.Expire(ctx, key, windowDurationSec); err != nil {
		logRedisError("EXPIRE", key, err)
		return false
	}

	return true
}

func isWithinLimit(count int64) bool {
	return count <= windowRequestLimit
}

func logRedisError(operation, key string, err error) {
	log.Printf("redis error (%s %s): %v", operation, key, err)
}

func currentMinuteWindow() string {
	return time.Now().UTC().Format(minuteWindowLayout)
}
