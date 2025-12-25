package redis

import (
	"context"
	"fmt"
	"os"

	"github.com/redis/go-redis/v9"
)

const servicePrefix = "ckd-epi."

type Config struct {
	Host     string
	Port     string
	Password string
}

type Client struct {
	client *redis.Client
}

func New(ctx context.Context, config Config) (*Client, error) {
	addr := fmt.Sprintf("%s:%s", config.Host, config.Port)
	
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: config.Password,
		DB:       0,
	})

	_, err := rdb.Ping(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &Client{
		client: rdb,
	}, nil
}

func NewFromEnv(ctx context.Context) (*Client, error) {
	config := Config{
		Host:     getEnv("REDIS_HOST", "localhost"),
		Port:     getEnv("REDIS_PORT", "6379"),
		Password: getEnv("REDIS_PASSWORD", ""),
	}
	return New(ctx, config)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}


