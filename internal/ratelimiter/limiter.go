package ratelimiter

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/Agentkma/distributed-rate-limiter/internal/redisclient"
	"github.com/redis/go-redis/v9"
)

const (
	windowRequestLimit = 3
	windowDurationSec  = 60 * time.Second
	requestTimeoutSec    = 2 * time.Second
	minuteWindowLayout   = "200601021504"
)

func Allow(clientAddress string) bool {
	ctx, cancel := newRequestContext()
	defer cancel()

	rateLimitKey := buildRateLimitKey(clientAddress)
	client := redisclient.GetClient()

	count, ok := incrementRequestCount(ctx, client, rateLimitKey)
	if !ok {
		// Fail-open: allow requests when Redis is unavailable.
		return true
	}

	if isFirstRequestForWindow(count) {
		if !setWindowExpiration(ctx, client, rateLimitKey) {
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

func incrementRequestCount(ctx context.Context, client *redis.Client, key string) (int64, bool) {
	count, err := client.Incr(ctx, key).Result()
	if err != nil {
		logRedisError("INCR", key, err)
		return 0, false
	}

	return count, true
}

func isFirstRequestForWindow(count int64) bool {
	return count == 1
}

func setWindowExpiration(ctx context.Context, client *redis.Client, key string) bool {
	if err := client.Expire(ctx, key, windowDurationSec).Err(); err != nil {
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
