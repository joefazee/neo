package cache

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/assert"
)

func setupRedisCache(t *testing.T, withOpTimeout time.Duration) (*RedisCache[string], *miniredis.Miniredis) {
	s, err := miniredis.Run()
	assert.NoError(t, err)
	opts := &RedisOptions{
		Addr:            s.Addr(),
		Password:        "",
		DB:              0,
		PoolSize:        5,
		MinIdleConns:    1,
		MaxRetries:      1,
		MinRetryBackoff: 1 * time.Millisecond,
		MaxRetryBackoff: 10 * time.Millisecond,
		OpTimeout:       withOpTimeout,
	}
	rc := NewRedisCache[string](opts)
	return rc, s
}

func TestRedisCacheDefaultOpTimeout_NoPanic(t *testing.T) {
	rc, s := setupRedisCache(t, 0)
	defer func() {
		rc.Close()
		s.Close()
	}()

	ctx := context.Background()
	assert.NoError(t, rc.Set(ctx, "foo", "bar", 0))
	v, err := rc.Get(ctx, "foo")
	assert.NoError(t, err)
	assert.Equal(t, "bar", v)
}

func TestRedisCacheBasicAndEdgeCases(t *testing.T) {
	s, err := miniredis.Run()
	assert.NoError(t, err)
	defer s.Close()

	opts := &RedisOptions{
		Addr:            s.Addr(),
		Password:        "",
		DB:              0,
		PoolSize:        10,
		MinIdleConns:    1,
		MaxRetries:      1,
		MinRetryBackoff: 1 * time.Millisecond,
		MaxRetryBackoff: 10 * time.Millisecond,
		OpTimeout:       100 * time.Millisecond,
	}
	rc := NewRedisCache[string](opts)
	defer rc.Close()
	ctx := context.Background()

	assert.NoError(t, rc.Set(ctx, "key", "value", 0))
	v, err := rc.Get(ctx, "key")
	assert.NoError(t, err)
	assert.Equal(t, "value", v)

	_, err = rc.Get(ctx, "missing")
	assert.ErrorIs(t, err, ErrCacheMiss)

	assert.NoError(t, rc.Set(ctx, "temp", "x", 50*time.Millisecond))
	s.FastForward(100 * time.Millisecond)
	v, err = rc.Get(ctx, "temp")
	assert.ErrorIs(t, err, ErrCacheMiss)
	assert.Empty(t, v)

	data := map[string]string{"a": "1", "b": "2"}
	assert.NoError(t, rc.MSet(ctx, data, 0))
	vals, errs := rc.MGet(ctx, "a", "b", "c")
	assert.Len(t, vals, 3)
	assert.Len(t, errs, 3)
	assert.NoError(t, errs[0])
	assert.Equal(t, "1", vals[0])
	assert.NoError(t, errs[1])
	assert.Equal(t, "2", vals[1])
	assert.ErrorIs(t, errs[2], ErrCacheMiss)

	assert.NoError(t, rc.Delete(ctx, "a"))
	_, err = rc.Get(ctx, "a")
	assert.ErrorIs(t, err, ErrCacheMiss)

	rc2 := NewRedisCache[string](opts)
	assert.NoError(t, rc2.Set(ctx, "foo", "bar", 0))
	assert.NoError(t, rc2.Close())
	_, err = rc2.Get(ctx, "foo")
	assert.Error(t, err)

	shortOpts := opts
	shortOpts.OpTimeout = time.Nanosecond
	rcShort := NewRedisCache[string](shortOpts)
	defer rcShort.Close()
	err = rcShort.Set(ctx, "k", "v", 0)
	assert.ErrorIs(t, err, context.DeadlineExceeded)

	rcFunc := NewRedisCache[func()](opts)
	defer rcFunc.Close()
	err = rcFunc.MSet(ctx, map[string]func(){"f": func() {}}, 0)
	assert.Error(t, err)
}

func TestRedisCacheMGetTypeAssertion(t *testing.T) {
	s, err := miniredis.Run()
	assert.NoError(t, err)
	defer s.Close()

	opts := &RedisOptions{
		Addr:         s.Addr(),
		Password:     "",
		DB:           0,
		PoolSize:     2,
		MinIdleConns: 0,
		MaxRetries:   0,
		OpTimeout:    100 * time.Millisecond,
	}
	rc := NewRedisCache[string](opts)
	defer rc.Close()
	ctx := context.Background()

	s.Set("bkey", "rawbytes")
	vals, errs := rc.MGet(ctx, "bkey", "nokey")
	assert.Len(t, vals, 2)
	assert.Len(t, errs, 2)

	assert.Error(t, errs[0])
	assert.Contains(t, errs[0].Error(), "invalid character")
	assert.Empty(t, vals[0])

	assert.ErrorIs(t, errs[1], ErrCacheMiss)
	assert.Empty(t, vals[1])
}

func TestRedisCacheGet_UnmarshalError(t *testing.T) {
	rc, s := setupRedisCache(t, 100*time.Millisecond)
	defer func() {
		rc.Close()
		s.Close()
	}()
	ctx := context.Background()

	s.Set("bad", "not-a-json")
	val, err := rc.Get(ctx, "bad")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid character")
	assert.Empty(t, val)
}

func TestRedisCacheSet_MarshalError(t *testing.T) {
	s, _ := miniredis.Run()
	defer s.Close()
	opts := &RedisOptions{
		Addr:         s.Addr(),
		Password:     "",
		DB:           0,
		PoolSize:     2,
		MinIdleConns: 1,
		OpTimeout:    50 * time.Millisecond,
	}
	rcFunc := NewRedisCache[func()](opts)
	defer rcFunc.Close()

	err := rcFunc.Set(context.Background(), "fn", func() {}, 0)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported type: func")
}

func TestRedisCacheMGet_UpstreamErrorPropagation(t *testing.T) {
	rc, s := setupRedisCache(t, 100*time.Millisecond)
	rc.Close()
	defer s.Close()

	vals, errs := rc.MGet(context.Background(), "x", "y", "z")
	assert.Len(t, vals, 3)
	assert.Len(t, errs, 3)
	for i, e := range errs {
		assert.Error(t, e, "expected error for index %d", i)
	}
	for _, v := range vals {
		assert.Empty(t, v)
	}
}

func TestRedisCacheMGet_BytesAndDefaultBranch(t *testing.T) {
	rc, s := setupRedisCache(t, 100*time.Millisecond)
	defer func() {
		rc.Close()
		s.Close()
	}()
	ctx := context.Background()

	err := rc.client.Set(ctx, "bkey", []byte(`"hello"`), 0).Err()
	assert.NoError(t, err)

	vals, errs := rc.MGet(ctx, "bkey")
	assert.Len(t, vals, 1)
	assert.Len(t, errs, 1)
	assert.NoError(t, errs[0])
	assert.Equal(t, "hello", vals[0])

	s.Set("not-hash", "foo")
	wrongTypeCmd := rc.client.HMGet(ctx, "not-hash", "field")
	_, hmErr := wrongTypeCmd.Result()
	assert.Error(t, hmErr)
}

func TestRedisCacheMSet_TTLBranch(t *testing.T) {
	rc, s := setupRedisCache(t, 100*time.Millisecond)
	defer func() {
		rc.Close()
		s.Close()
	}()
	ctx := context.Background()

	ttl := 50 * time.Millisecond
	data := map[string]string{"x": "1", "y": "2"}
	assert.NoError(t, rc.MSet(ctx, data, ttl))

	s.FastForward(ttl + 10*time.Millisecond)

	vals, errs := rc.MGet(ctx, "x", "y")
	assert.Len(t, vals, 2)
	assert.Len(t, errs, 2)
	assert.ErrorIs(t, errs[0], ErrCacheMiss)
	assert.ErrorIs(t, errs[1], ErrCacheMiss)
}
