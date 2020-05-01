package rproxy

import (
	"context"
	"golang.org/x/crypto/acme/autocert"
	"sync"
)

type memCache struct {
	lock sync.RWMutex
	data map[string][]byte
}

func (m *memCache) Get(ctx context.Context, key string) ([]byte, error) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	if result, ok := m.data[key]; ok {
		return result, nil
	}

	return nil, autocert.ErrCacheMiss
}

func (m *memCache) Put(_ context.Context, key string, data []byte) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	m.data[key] = data

	return nil
}

func (m *memCache) Delete(ctx context.Context, key string) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	delete(m.data, key)

	return nil
}

func NewMemCache() autocert.Cache {
	return &memCache{data: map[string][]byte{}}
}
