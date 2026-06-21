package ratelimiter

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/Agentkma/distributed-rate-limiter/internal/redisclient"
)

const (
	limit         = 3
	windowSeconds = 60
)

func Allow(ip string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	key := fmt.Sprintf("rate:%s:%s", ip, currentMinuteBucket())

	client := redisclient.GetClient()
	count, err := client.Incr(ctx, key).Result()
	if err != nil {
		log.Printf("redis error (INCR %s): %v", key, err)
		return true
	}

	if count == 1 {
		if err := client.Expire(ctx, key, windowSeconds*time.Second).Err(); err != nil {
			log.Printf("redis error (EXPIRE %s): %v", key, err)
			return true
		}
	}

	return count <= limit
}

func currentMinuteBucket() string {
	return time.Now().UTC().Format("200601021504")
}
