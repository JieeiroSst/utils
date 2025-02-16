package fifo

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/redis/go-redis/v9"
)

type FIFOCache struct {
	client    *redis.Client
	hashKey   string
	maxItems  int
	itemOrder []string 
}

func NewFIFOCache(client *redis.Client, hashKey string, maxItems int) *FIFOCache {
	return &FIFOCache{
		client:    client,
		hashKey:   hashKey,
		maxItems:  maxItems,
		itemOrder: make([]string, 0),
	}
}

func (c *FIFOCache) Set(ctx context.Context, key string, value interface{}) error {
	jsonValue, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}

	pipe := c.client.Pipeline()

	pipe.HSet(ctx, c.hashKey, key, jsonValue)

	c.itemOrder = append(c.itemOrder, key)

	if len(c.itemOrder) > c.maxItems {
		oldestKey := c.itemOrder[0]
		c.itemOrder = c.itemOrder[1:]
		pipe.HDel(ctx, c.hashKey, oldestKey)
	}

	_, err = pipe.Exec(ctx)
	return err
}

func (c *FIFOCache) Get(ctx context.Context, key string) (interface{}, error) {
	value, err := c.client.HGet(ctx, c.hashKey, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("key not found")
		}
		return nil, err
	}

	var result interface{}
	err = json.Unmarshal([]byte(value), &result)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal value: %w", err)
	}

	return result, nil
}

func (c *FIFOCache) Delete(ctx context.Context, key string) error {
	err := c.client.HDel(ctx, c.hashKey, key).Err()
	if err != nil {
		return err
	}

	for i, k := range c.itemOrder {
		if k == key {
			c.itemOrder = append(c.itemOrder[:i], c.itemOrder[i+1:]...)
			break
		}
	}

	return nil
}

func (c *FIFOCache) Size(ctx context.Context) int64 {
	return c.client.HLen(ctx, c.hashKey).Val()
}
