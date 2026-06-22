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
	maxRequestsPerWindow = 3
	windowDuration       = 60 * time.Second
	requestTimeout       = 2 * time.Second
)

func Allow(clientAddress string) bool {
	ctx, cancel := newRequestContext()
	defer cancel()

	key := buildRateLimitKey(clientAddress)
	client := redisclient.GetClient()

	count, ok := incrementRequestCount(ctx, client, key)
	if !ok {
		return true
	}

	if isFirstRequestInWindow(count) {
		if !setWindowExpiration(ctx, client, key) {
			return true
		}
	}

	return isWithinLimit(count)
}

func newRequestContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), requestTimeout)
}

func buildRateLimitKey(clientAddress string) string {
	return fmt.Sprintf("rate:%s:%s", clientAddress, currentMinuteBucket())
}

func incrementRequestCount(ctx context.Context, client *redis.Client, key string) (int64, bool) {
	count, err := client.Incr(ctx, key).Result()
	if err != nil {
		logRedisError("INCR", key, err)
		return 0, false
	}

	return count, true
}

func isFirstRequestInWindow(count int64) bool {
	return count == 1
}

func setWindowExpiration(ctx context.Context, client *redis.Client, key string) bool {
	if err := client.Expire(ctx, key, windowDuration).Err(); err != nil {
		logRedisError("EXPIRE", key, err)
		return false
	}

	return true
}

func isWithinLimit(count int64) bool {
	return count <= maxRequestsPerWindow
}

func logRedisError(operation, key string, err error) {
	log.Printf("redis error (%s %s): %v", operation, key, err)
}

func currentMinuteBucket() string {
	return time.Now().UTC().Format("200601021504")
}
