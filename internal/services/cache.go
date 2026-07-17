package services

// CacheClearer removes all locally cached data.
// Implemented by *cache.Store.
type CacheClearer interface {
	Clear() error
}

// CacheService exposes cache maintenance operations.
type CacheService struct {
	store CacheClearer
}

// NewCacheService wires the cache store dependency.
func NewCacheService(store CacheClearer) *CacheService {
	return &CacheService{store: store}
}

// Clear removes all cached API responses.
func (s *CacheService) Clear() error {
	return s.store.Clear()
}
