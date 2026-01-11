package cache

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/go-redis/redis/v8"
)

type Redis struct {
	client *redis.Client
	ctx    context.Context
}

func NewRedis() (*Redis, error) {
	host := getEnv("REDIS_HOST", "localhost")
	port := getEnv("REDIS_PORT", "6379")

	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", host, port),
		Password: "",
		DB:       0,
	})

	ctx := context.Background()

	// Verify connection
	_, err := client.Ping(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	log.Println("Redis connected successfully")
	return &Redis{client: client, ctx: ctx}, nil
}

// Set stores a key-value pair with expiration
func (r *Redis) Set(key string, value string, expiration time.Duration) error {
	return r.client.Set(r.ctx, key, value, expiration).Err()
}

// Get retrieves a value by key
func (r *Redis) Get(key string) (string, error) {
	return r.client.Get(r.ctx, key).Result()
}

// Exists checks if a key exists
func (r *Redis) Exists(key string) (bool, error) {
	result, err := r.client.Exists(r.ctx, key).Result()
	if err != nil {
		return false, err
	}
	return result > 0, nil
}

// Incr increments a key's value
func (r *Redis) Incr(key string) (int64, error) {
	return r.client.Incr(r.ctx, key).Result()
}

// Expire sets expiration on a key
func (r *Redis) Expire(key string, expiration time.Duration) error {
	return r.client.Expire(r.ctx, key, expiration).Err()
}

// Close closes the Redis connection
func (r *Redis) Close() error {
	return r.client.Close()
}

// Ping checks Redis connection
func (r *Redis) Ping() error {
	_, err := r.client.Ping(r.ctx).Result()
	return err
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
