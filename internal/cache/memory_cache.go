package cache

import (
	"context"
	"sync"
	"time"
)

type item[V any] struct {
	value      V
	expiration int64 // Unix nanoseconds; zero = no expire
}

type shard[V any] struct {
	sync.RWMutex
	items map[string]item[V]
}

type MemoryCache[V any] struct {
	shards []*shard[V]
	quit   chan struct{}
}

// NewMemoryCache creates a 256-shard cache with a 1s janitor by default.
func NewMemoryCache[V any]() *MemoryCache[V] {
	return NewMemoryCacheWithOptions[V](256, 1*time.Second)
}

// NewMemoryCacheWithOptions allows customizing shard count & janitor interval.
func NewMemoryCacheWithOptions[V any](shardCount int, janitorInterval time.Duration) *MemoryCache[V] {
	mc := &MemoryCache[V]{
		shards: make([]*shard[V], shardCount),
		quit:   make(chan struct{}),
	}
	for i := 0; i < shardCount; i++ {
		mc.shards[i] = &shard[V]{items: make(map[string]item[V])}
	}
	go mc.startJanitor(janitorInterval)
	return mc
}

// Stop terminates the janitor goroutine and releases resources.
func (mc *MemoryCache[V]) Stop() {
	select {
	case <-mc.quit:
	default:
		close(mc.quit)
	}
}

func (mc *MemoryCache[V]) getShard(key string) *shard[V] {
	h := fnv32(key)
	return mc.shards[int(h)%len(mc.shards)]
}

func fnv32(key string) uint32 {
	const offset = 2166136261
	const prime = 16777619
	h := uint32(offset)
	for i := 0; i < len(key); i++ {
		h ^= uint32(key[i])
		h *= prime
	}
	return h
}

// Get does an atomic lock/unlock to avoid the RLock→Lock race.
func (mc *MemoryCache[V]) Get(_ context.Context, key string) (V, error) {
	var zero V
	now := time.Now().UnixNano()
	s := mc.getShard(key)

	s.Lock()
	itm, ok := s.items[key]
	if ok {
		if itm.expiration > 0 && now > itm.expiration {
			delete(s.items, key)
			ok = false
		}
	}
	if ok {
		val := itm.value
		s.Unlock()
		return val, nil
	}
	s.Unlock()
	return zero, ErrCacheMiss
}

func (mc *MemoryCache[V]) Set(_ context.Context, key string, value V, ttl time.Duration) error {
	var exp int64
	if ttl > 0 {
		exp = time.Now().Add(ttl).UnixNano()
	}
	s := mc.getShard(key)
	s.Lock()
	s.items[key] = item[V]{value: value, expiration: exp}
	s.Unlock()
	return nil
}

func (mc *MemoryCache[V]) Delete(_ context.Context, key string) error {
	s := mc.getShard(key)
	s.Lock()
	delete(s.items, key)
	s.Unlock()
	return nil
}

func (mc *MemoryCache[V]) MGet(_ context.Context, keys ...string) ([]V, []error) {
	results := make([]V, len(keys))
	errs := make([]error, len(keys))

	type req struct {
		idx int
		key string
	}
	groups := make(map[*shard[V]][]req, len(mc.shards))
	for i, k := range keys {
		sh := mc.getShard(k)
		groups[sh] = append(groups[sh], req{i, k})
	}

	for sh, reqs := range groups {
		now := time.Now().UnixNano()
		sh.Lock()
		for _, r := range reqs {
			itm, ok := sh.items[r.key]
			if !ok || (itm.expiration > 0 && now > itm.expiration) {
				if ok {
					delete(sh.items, r.key)
				}
				errs[r.idx] = ErrCacheMiss
			} else {
				results[r.idx] = itm.value
			}
		}
		sh.Unlock()
	}
	return results, errs
}

func (mc *MemoryCache[V]) MSet(_ context.Context, kv map[string]V, ttl time.Duration) error {
	var exp int64
	if ttl > 0 {
		exp = time.Now().Add(ttl).UnixNano()
	}
	groups := make(map[*shard[V]]map[string]item[V], len(mc.shards))
	for k, v := range kv {
		sh := mc.getShard(k)
		if groups[sh] == nil {
			groups[sh] = make(map[string]item[V])
		}
		groups[sh][k] = item[V]{value: v, expiration: exp}
	}
	for sh, entries := range groups {
		sh.Lock()
		for k, itm := range entries {
			sh.items[k] = itm
		}
		sh.Unlock()
	}
	return nil
}

func (mc *MemoryCache[V]) startJanitor(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			now := time.Now().UnixNano()
			for _, sh := range mc.shards {
				go func(s *shard[V]) {
					s.Lock()
					for k, itm := range s.items {
						if itm.expiration > 0 && now > itm.expiration {
							delete(s.items, k)
						}
					}
					s.Unlock()
				}(sh)
			}
		case <-mc.quit:
			return
		}
	}
}
