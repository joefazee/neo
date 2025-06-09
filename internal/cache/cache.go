package cache

import (
	"context"
	"errors"
	"time"
)

const (
	RedisBackend  = "redis"
	MemoryBackend = "memory"
)

var ErrCacheMiss = errors.New("cache: key not found")

// Cache is our generic cache interface.
type Cache[V any] interface {
	// Get returns the value or ErrCacheMiss.
	Get(ctx context.Context, key string) (V, error)
	// Set stores value under key, with TTL. Zero ttl = no expiration.
	Set(ctx context.Context, key string, value V, ttl time.Duration) error
	// Delete removes the key.
	Delete(ctx context.Context, key string) error
	// MGet returns multiple values; missing ones are zero-value + ErrCacheMiss.
	MGet(ctx context.Context, keys ...string) ([]V, []error)
	// MSet sets multiple key/value pairs with same TTL.
	MSet(ctx context.Context, kv map[string]V, ttl time.Duration) error
}

func NewCache[V any](backend string, opts ...interface{}) Cache[V] {
	switch backend {
	case RedisBackend:
		return NewRedisCache[V](opts[0].(*RedisOptions))
	case MemoryBackend:
		return NewMemoryCache[V]()
	default:
		panic("unknown cache backend")
	}
}
