package redisclient

import (
	"sync"

	"github.com/redis/go-redis/v9"
)

const localRedisAddr = "localhost:6379"

var (
	client *redis.Client
	once   sync.Once
)

func GetClient() *redis.Client {
	once.Do(func() {
		client = redis.NewClient(&redis.Options{
			Addr: localRedisAddr,
		})
	})

	return client
}
