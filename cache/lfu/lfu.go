package lfu

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
)

type Cache struct {
	client  *redis.Client
	hashKey string
}

func NewCache(addr, password string, hashKey string) (*Cache, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       0,
	})

	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	if err := client.ConfigSet(ctx, "maxmemory", "100mb").Err(); err != nil {
		return nil, err
	}

	if err := client.ConfigSet(ctx, "maxmemory-policy", "allkeys-lfu").Err(); err != nil {
		return nil, err
	}

	return &Cache{
		client:  client,
		hashKey: hashKey,
	}, nil
}

func (c *Cache) Set(ctx context.Context, field string, value interface{}, expiration time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	if err := c.client.HSet(ctx, c.hashKey, field, data).Err(); err != nil {
		return err
	}

	if expiration > 0 {
		return c.client.Expire(ctx, c.hashKey, expiration).Err()
	}

	return nil
}

func (c *Cache) Get(ctx context.Context, field string, dest interface{}) error {
	data, err := c.client.HGet(ctx, c.hashKey, field).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil
		}
		return err
	}

	return json.Unmarshal(data, dest)
}

func (c *Cache) Delete(ctx context.Context, field string) error {
	return c.client.HDel(ctx, c.hashKey, field).Err()
}

func (c *Cache) Clear(ctx context.Context) error {
	return c.client.Del(ctx, c.hashKey).Err()
}

func (c *Cache) GetAll(ctx context.Context) (map[string]string, error) {
	return c.client.HGetAll(ctx, c.hashKey).Result()
}

func (c *Cache) Close() error {
	return c.client.Close()
}
