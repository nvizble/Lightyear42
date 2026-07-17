package repository

import (
	"context"
	"encoding/json"
	"strings"
	"time"
)

// cacheVersion prefixes every cache key. Bump it whenever the serialized
// shape of cached models changes, so stale entries from older binaries are
// ignored instead of being decoded with missing fields.
const cacheVersion = "v1"

// cacheKey builds a namespaced, versioned cache key.
func cacheKey(parts ...string) string {
	return cacheVersion + ":" + strings.Join(parts, ":")
}

// KVCache is the minimal cache surface used by repositories.
// Implemented by *cache.Store.
type KVCache interface {
	Get(key string) ([]byte, bool, error)
	Set(key string, value []byte, ttl time.Duration) error
}

// NoopCache disables caching (every read goes to the API).
type NoopCache struct{}

// Get always misses.
func (NoopCache) Get(string) ([]byte, bool, error) { return nil, false, nil }

// Set discards the value.
func (NoopCache) Set(string, []byte, time.Duration) error { return nil }

// fetchCached implements the read-through pattern: return the cached value
// when fresh, otherwise fetch, cache and return.
//
// Cache failures never break the request: a broken local cache must not
// prevent API access, so errors from Get/Set are deliberately swallowed.
func fetchCached[T any](ctx context.Context, kv KVCache, key string, ttl time.Duration, fetch func(context.Context) (T, error)) (T, error) {
	if data, hit, err := kv.Get(key); err == nil && hit {
		var value T
		if err := json.Unmarshal(data, &value); err == nil {
			return value, nil
		}
	}

	value, err := fetch(ctx)
	if err != nil {
		var zero T
		return zero, err
	}

	if data, err := json.Marshal(value); err == nil {
		_ = kv.Set(key, data, ttl)
	}
	return value, nil
}
