package cache

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/assert"
)

func TestNewCacheMemory(t *testing.T) {
	c := NewCache[string]("memory")
	m, ok := c.(*MemoryCache[string])
	assert.True(t, ok, "expected *MemoryCache[string]")
	ctx := context.Background()

	err := m.Set(ctx, "foo", "bar", 0)
	assert.NoError(t, err)
	v, err := m.Get(ctx, "foo")
	assert.NoError(t, err)
	assert.Equal(t, "bar", v)

	_, err = m.Get(ctx, "missing")
	assert.ErrorIs(t, err, ErrCacheMiss)
}

func TestNewCacheRedis(t *testing.T) {
	s, err := miniredis.Run()
	assert.NoError(t, err)
	defer s.Close()

	opts := RedisOptions{
		Addr:            s.Addr(),
		Password:        "",
		DB:              0,
		PoolSize:        5,
		MinIdleConns:    1,
		MaxRetries:      0,
		MinRetryBackoff: 1 * time.Millisecond,
		MaxRetryBackoff: 1 * time.Millisecond,
		OpTimeout:       100 * time.Millisecond,
	}
	c := NewCache[string]("redis", &opts)
	r, ok := c.(*RedisCache[string])
	assert.True(t, ok, "expected *RedisCache[string]")
	defer r.Close()
	ctx := context.Background()

	err = r.Set(ctx, "foo", "baz", 0)
	assert.NoError(t, err)
	v, err := r.Get(ctx, "foo")
	assert.NoError(t, err)
	assert.Equal(t, "baz", v)

	_, err = r.Get(ctx, "missing")
	assert.ErrorIs(t, err, ErrCacheMiss)
}

func TestNewCacheUnknownPanics(t *testing.T) {
	assert.Panics(t, func() {
		_ = NewCache[int]("something-else")
	})
}
