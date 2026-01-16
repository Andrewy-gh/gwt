package cache

import (
	"sync"
	"time"
)

// cacheItem represents a single item in the cache with its expiration time
type cacheItem[V any] struct {
	value      V
	expiration time.Time
}

// isExpired checks if the cache item has expired
func (item *cacheItem[V]) isExpired() bool {
	return time.Now().After(item.expiration)
}

// Cache is a generic thread-safe cache with TTL support
type Cache[K comparable, V any] struct {
	items map[K]*cacheItem[V]
	mu    sync.RWMutex
	ttl   time.Duration
}

// New creates a new cache with the specified TTL
func New[K comparable, V any](ttl time.Duration) *Cache[K, V] {
	c := &Cache[K, V]{
		items: make(map[K]*cacheItem[V]),
		ttl:   ttl,
	}

	// Start background cleanup goroutine
	go c.cleanup()

	return c
}

// Get retrieves a value from the cache
// Returns the value and true if found and not expired, zero value and false otherwise
func (c *Cache[K, V]) Get(key K) (V, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, exists := c.items[key]
	if !exists {
		var zero V
		return zero, false
	}

	if item.isExpired() {
		var zero V
		return zero, false
	}

	return item.value, true
}

// Set stores a value in the cache with the default TTL
func (c *Cache[K, V]) Set(key K, value V) {
	c.SetWithTTL(key, value, c.ttl)
}

// SetWithTTL stores a value in the cache with a custom TTL
func (c *Cache[K, V]) SetWithTTL(key K, value V, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items[key] = &cacheItem[V]{
		value:      value,
		expiration: time.Now().Add(ttl),
	}
}

// Delete removes a key from the cache
func (c *Cache[K, V]) Delete(key K) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.items, key)
}

// Clear removes all items from the cache
func (c *Cache[K, V]) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items = make(map[K]*cacheItem[V])
}

// Len returns the number of items in the cache (including expired items)
func (c *Cache[K, V]) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.items)
}

// cleanup periodically removes expired items from the cache
func (c *Cache[K, V]) cleanup() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		c.removeExpired()
	}
}

// removeExpired removes all expired items from the cache
func (c *Cache[K, V]) removeExpired() {
	c.mu.Lock()
	defer c.mu.Unlock()

	for key, item := range c.items {
		if item.isExpired() {
			delete(c.items, key)
		}
	}
}

// GetOrSet retrieves a value from the cache, or sets it if not found
// The loader function is called only if the key is not found or expired
func (c *Cache[K, V]) GetOrSet(key K, loader func() (V, error)) (V, error) {
	// Try to get from cache first
	if value, found := c.Get(key); found {
		return value, nil
	}

	// Load the value
	value, err := loader()
	if err != nil {
		var zero V
		return zero, err
	}

	// Store in cache
	c.Set(key, value)

	return value, nil
}

// InvalidatePattern removes all keys matching the provided pattern function
// This is useful for invalidating related cache entries
func (c *Cache[K, V]) InvalidatePattern(matcher func(K) bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for key := range c.items {
		if matcher(key) {
			delete(c.items, key)
		}
	}
}

// Keys returns all keys in the cache (including expired items)
func (c *Cache[K, V]) Keys() []K {
	c.mu.RLock()
	defer c.mu.RUnlock()

	keys := make([]K, 0, len(c.items))
	for key := range c.items {
		keys = append(keys, key)
	}

	return keys
}

// Stats returns statistics about the cache
type Stats struct {
	TotalItems   int
	ExpiredItems int
}

// GetStats returns current cache statistics
func (c *Cache[K, V]) GetStats() Stats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	stats := Stats{
		TotalItems: len(c.items),
	}

	for _, item := range c.items {
		if item.isExpired() {
			stats.ExpiredItems++
		}
	}

	return stats
}

// SetTTL updates the default TTL for the cache
// This does not affect existing items
func (c *Cache[K, V]) SetTTL(ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.ttl = ttl
}

// GetTTL returns the current default TTL
func (c *Cache[K, V]) GetTTL() time.Duration {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.ttl
}
