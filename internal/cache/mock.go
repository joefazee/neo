package cache

import (
	"context"
	"time"

	"github.com/stretchr/testify/mock"
)

type MockCache struct {
	mock.Mock
}

func (m *MockCache) Get(ctx context.Context, key string) (string, error) {
	args := m.Called(ctx, key)
	return args.String(0), args.Error(1)
}

func (m *MockCache) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	args := m.Called(ctx, key, value, ttl)
	return args.Error(0)
}

func (m *MockCache) Delete(ctx context.Context, key string) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

func (m *MockCache) MGet(ctx context.Context, keys ...string) ([]string, []error) {
	args := m.Called(ctx, keys)
	return args.Get(0).([]string), args.Get(1).([]error)
}

func (m *MockCache) MSet(ctx context.Context, kv map[string]string, ttl time.Duration) error {
	args := m.Called(ctx, kv, ttl)
	return args.Error(0)
}
