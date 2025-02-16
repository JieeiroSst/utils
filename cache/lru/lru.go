package lru

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type LRUCache struct {
	client     *redis.Client
	hashKey    string
	maxEntries int
}

type CacheEntry struct {
	Value     interface{} `json:"value"`
	Timestamp int64       `json:"timestamp"`
}

func NewLRUCache(client *redis.Client, hashKey string, maxEntries int) *LRUCache {
	return &LRUCache{
		client:     client,
		hashKey:    hashKey,
		maxEntries: maxEntries,
	}
}

func (c *LRUCache) Set(ctx context.Context, key string, value interface{}) error {
	entry := CacheEntry{
		Value:     value,
		Timestamp: time.Now().UnixNano(),
	}

	serialized, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal cache entry: %w", err)
	}

	size, err := c.client.HLen(ctx, c.hashKey).Result()
	if err != nil {
		return fmt.Errorf("failed to get hash length: %w", err)
	}

	if int(size) >= c.maxEntries {
		if err := c.removeOldest(ctx); err != nil {
			return fmt.Errorf("failed to remove oldest entry: %w", err)
		}
	}

	return c.client.HSet(ctx, c.hashKey, key, serialized).Err()
}

func (c *LRUCache) Get(ctx context.Context, key string) (interface{}, error) {
	serialized, err := c.client.HGet(ctx, c.hashKey, key).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get cache entry: %w", err)
	}

	var entry CacheEntry
	if err := json.Unmarshal([]byte(serialized), &entry); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cache entry: %w", err)
	}

	entry.Timestamp = time.Now().UnixNano()
	updated, err := json.Marshal(entry)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal updated entry: %w", err)
	}

	if err := c.client.HSet(ctx, c.hashKey, key, updated).Err(); err != nil {
		return nil, fmt.Errorf("failed to update entry timestamp: %w", err)
	}

	return entry.Value, nil
}

func (c *LRUCache) removeOldest(ctx context.Context) error {
	entries, err := c.client.HGetAll(ctx, c.hashKey).Result()
	if err != nil {
		return fmt.Errorf("failed to get all entries: %w", err)
	}

	var oldestKey string
	var oldestTime int64 = time.Now().UnixNano()

	for key, serialized := range entries {
		var entry CacheEntry
		if err := json.Unmarshal([]byte(serialized), &entry); err != nil {
			continue
		}
		if entry.Timestamp < oldestTime {
			oldestTime = entry.Timestamp
			oldestKey = key
		}
	}

	if oldestKey != "" {
		return c.client.HDel(ctx, c.hashKey, oldestKey).Err()
	}

	return nil
}

func (c *LRUCache) Clear(ctx context.Context) error {
	return c.client.Del(ctx, c.hashKey).Err()
}
