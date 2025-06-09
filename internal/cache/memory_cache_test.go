package cache

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMemoryCacheBasicAndEdgeCases(t *testing.T) {
	mc := NewMemoryCache[string]()
	defer mc.Stop()
	ctx := context.Background()

	assert.NoError(t, mc.Set(ctx, "key", "value", 0))
	v, err := mc.Get(ctx, "key")
	assert.NoError(t, err)
	assert.Equal(t, "value", v)

	_, err = mc.Get(ctx, "missing")
	assert.ErrorIs(t, err, ErrCacheMiss)

	assert.NoError(t, mc.Set(ctx, "temp", "x", 50*time.Millisecond))
	time.Sleep(100 * time.Millisecond)
	_, err = mc.Get(ctx, "temp")
	assert.ErrorIs(t, err, ErrCacheMiss)

	data := map[string]string{"a": "1", "b": "2"}
	assert.NoError(t, mc.MSet(ctx, data, 0))
	vals, errs := mc.MGet(ctx, "a", "b", "c")
	assert.Len(t, vals, 3)
	assert.Len(t, errs, 3)
	assert.NoError(t, errs[0])
	assert.Equal(t, "1", vals[0])
	assert.NoError(t, errs[1])
	assert.Equal(t, "2", vals[1])
	assert.ErrorIs(t, errs[2], ErrCacheMiss)

	assert.NoError(t, mc.Delete(ctx, "a"))
	_, err = mc.Get(ctx, "a")
	assert.ErrorIs(t, err, ErrCacheMiss)

	assert.NoError(t, mc.MSet(ctx, map[string]string{"x": "y"}, 50*time.Millisecond))
	time.Sleep(100 * time.Millisecond)
	vs, es := mc.MGet(ctx, "x")
	assert.Len(t, vs, 1)
	assert.Len(t, es, 1)
	assert.ErrorIs(t, es[0], ErrCacheMiss)
}

func TestMemoryCacheCustomShardCount(t *testing.T) {
	mc := NewMemoryCacheWithOptions[int](4, time.Hour)
	defer mc.Stop()
	ctx := context.Background()

	for i := 0; i < 10; i++ {
		key := fmt.Sprintf("key%d", i)
		assert.NoError(t, mc.Set(ctx, key, i, 0))
		v, err := mc.Get(ctx, key)
		assert.NoError(t, err)
		assert.Equal(t, i, v)
	}
}

func TestMemoryCacheStopIdempotent(t *testing.T) {
	mc := NewMemoryCache[string]()
	assert.NotPanics(t, func() {
		mc.Stop()
		mc.Stop()
	})
}

func TestMemoryCacheConcurrency(t *testing.T) {
	assert.NotPanics(t, func() {
		mc := NewMemoryCache[int]()
		defer mc.Stop()
		ctx := context.Background()

		done := make(chan struct{})
		go func() {
			for i := 0; i < 1000; i++ {
				mc.Set(ctx, fmt.Sprintf("key%d", i), i, 0)
			}
			close(done)
		}()
		for i := 0; i < 1000; i++ {
			go mc.Get(ctx, fmt.Sprintf("key%d", i))
		}
		<-done
	})
}

func TestMemoryCacheJanitorCleansExpiredEntries(t *testing.T) {
	interval := 10 * time.Millisecond
	mc := NewMemoryCacheWithOptions[string](4, interval)
	defer mc.Stop()
	ctx := context.Background()

	ttl := 20 * time.Millisecond
	err := mc.Set(ctx, "to_clean", "value", ttl)
	assert.NoError(t, err, "Set should succeed")

	time.Sleep(ttl + interval + 10*time.Millisecond)

	found := false
	for _, shard := range mc.shards {
		shard.RLock()
		if _, ok := shard.items["to_clean"]; ok {
			found = true
		}
		shard.RUnlock()
	}
	assert.False(t, found, "expired entry should have been removed by janitor")
}
