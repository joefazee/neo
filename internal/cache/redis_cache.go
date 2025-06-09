package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisOptions holds both client‐tuning and operation‐level settings.
type RedisOptions struct {
	Addr            string
	Password        string
	DB              int
	PoolSize        int // e.g. 1000
	MinIdleConns    int
	MaxRetries      int           // retry count for transient errors
	MinRetryBackoff time.Duration // e.g. 8 * time.Millisecond
	MaxRetryBackoff time.Duration // e.g. 512 * time.Millisecond
	OpTimeout       time.Duration // per‐call timeout; defaulted if zero
}

type RedisCache[V any] struct {
	client    *redis.Client
	opTimeout time.Duration
}

// NewRedisCache constructs and configures the client (including backoff/retries) and default timeouts.
func NewRedisCache[V any](opts *RedisOptions) *RedisCache[V] {
	if opts.OpTimeout == 0 {
		opts.OpTimeout = 50 * time.Millisecond
	}
	client := redis.NewClient(&redis.Options{
		Addr:            opts.Addr,
		Password:        opts.Password,
		DB:              opts.DB,
		PoolSize:        opts.PoolSize,
		MinIdleConns:    opts.MinIdleConns,
		MaxRetries:      opts.MaxRetries,
		MinRetryBackoff: opts.MinRetryBackoff,
		MaxRetryBackoff: opts.MaxRetryBackoff,
	})
	return &RedisCache[V]{
		client:    client,
		opTimeout: opts.OpTimeout,
	}
}

// Close cleans up underlying connections.
func (r *RedisCache[V]) Close() error {
	return r.client.Close()
}

func (r *RedisCache[V]) Get(ctx context.Context, key string) (V, error) {
	var zero V
	// enforce timeout
	ctx, cancel := context.WithTimeout(ctx, r.opTimeout)
	defer cancel()

	data, err := r.client.Get(ctx, key).Bytes()
	if errors.Is(err, redis.Nil) {
		return zero, ErrCacheMiss
	} else if err != nil {
		return zero, err
	}
	var val V
	if err := json.Unmarshal(data, &val); err != nil {
		return zero, err
	}
	return val, nil
}

func (r *RedisCache[V]) Set(ctx context.Context, key string, value V, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, r.opTimeout)
	defer cancel()

	if ttl > 0 {
		return r.client.Set(ctx, key, data, ttl).Err()
	}
	return r.client.Set(ctx, key, data, 0).Err()
}

func (r *RedisCache[V]) Delete(ctx context.Context, key string) error {
	ctx, cancel := context.WithTimeout(ctx, r.opTimeout)
	defer cancel()
	return r.client.Del(ctx, key).Err()
}

func (r *RedisCache[V]) MGet(ctx context.Context, keys ...string) ([]V, []error) {
	results := make([]V, len(keys))
	errs := make([]error, len(keys))

	ctx, cancel := context.WithTimeout(ctx, r.opTimeout)
	defer cancel()

	vals, err := r.client.MGet(ctx, keys...).Result()
	if err != nil {
		for i := range errs {
			errs[i] = err
		}
		return results, errs
	}

	for i, raw := range vals {
		if raw == nil {
			errs[i] = ErrCacheMiss
			continue
		}

		var data []byte
		switch v := raw.(type) {
		case string:
			data = []byte(v)
		case []byte:
			data = v
		default:
			errs[i] = fmt.Errorf("unexpected type %T from redis", v)
			continue
		}

		var val V
		if err := json.Unmarshal(data, &val); err != nil {
			errs[i] = err
		} else {
			results[i] = val
		}
	}
	return results, errs
}

func (r *RedisCache[V]) MSet(ctx context.Context, kv map[string]V, ttl time.Duration) error {
	type entry struct {
		key  string
		data []byte
	}
	entries := make([]entry, 0, len(kv))
	for k, v := range kv {
		b, err := json.Marshal(v)
		if err != nil {
			return err
		}
		entries = append(entries, entry{key: k, data: b})
	}

	ctx, cancel := context.WithTimeout(ctx, r.opTimeout)
	defer cancel()

	pipe := r.client.Pipeline()
	for _, e := range entries {
		if ttl > 0 {
			pipe.Set(ctx, e.key, e.data, ttl)
		} else {
			pipe.Set(ctx, e.key, e.data, 0)
		}
	}
	_, err := pipe.Exec(ctx)
	return err
}
