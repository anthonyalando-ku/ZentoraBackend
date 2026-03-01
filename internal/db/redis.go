// internal/db/redis.go
package db

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisConfig struct {
	ClusterMode bool
	Addresses   []string
	Password    string
	DB          int
	PoolSize    int
}

func NewRedisClusterClient(cfg RedisConfig) (*redis.ClusterClient, error) {
	client := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs:    cfg.Addresses,
		Password: cfg.Password,
		PoolSize: cfg.PoolSize,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis cluster: %w", err)
	}

	return client, nil
}

// For single node Redis (development)
func NewRedisClient(cfg RedisConfig) (*redis.Client, error) {
	if len(cfg.Addresses) == 0 {
		return nil, fmt.Errorf("no Redis address provided")
	}

	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addresses[0],
		Password: cfg.Password,
		DB:       cfg.DB,
		PoolSize: cfg.PoolSize,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return client, nil
}