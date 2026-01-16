package cache

import (
	"sync"
	"testing"
	"time"
)

func TestCacheBasicOperations(t *testing.T) {
	c := New[string, string](time.Minute)

	// Test Set and Get
	c.Set("key1", "value1")
	value, found := c.Get("key1")
	if !found {
		t.Error("Expected to find key1")
	}
	if value != "value1" {
		t.Errorf("Expected value1, got %s", value)
	}

	// Test non-existent key
	_, found = c.Get("nonexistent")
	if found {
		t.Error("Expected not to find nonexistent key")
	}

	// Test Delete
	c.Delete("key1")
	_, found = c.Get("key1")
	if found {
		t.Error("Expected key1 to be deleted")
	}
}

func TestCacheTTL(t *testing.T) {
	// Create cache with very short TTL
	c := New[string, string](100 * time.Millisecond)

	c.Set("key1", "value1")

	// Should be found immediately
	value, found := c.Get("key1")
	if !found {
		t.Error("Expected to find key1 immediately")
	}
	if value != "value1" {
		t.Errorf("Expected value1, got %s", value)
	}

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Should not be found after expiration
	_, found = c.Get("key1")
	if found {
		t.Error("Expected key1 to be expired")
	}
}

func TestCacheSetWithTTL(t *testing.T) {
	c := New[string, string](time.Minute)

	// Set with custom short TTL
	c.SetWithTTL("key1", "value1", 100*time.Millisecond)

	// Should be found immediately
	_, found := c.Get("key1")
	if !found {
		t.Error("Expected to find key1 immediately")
	}

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Should not be found after expiration
	_, found = c.Get("key1")
	if found {
		t.Error("Expected key1 to be expired")
	}
}

func TestCacheClear(t *testing.T) {
	c := New[string, string](time.Minute)

	c.Set("key1", "value1")
	c.Set("key2", "value2")
	c.Set("key3", "value3")

	if c.Len() != 3 {
		t.Errorf("Expected length 3, got %d", c.Len())
	}

	c.Clear()

	if c.Len() != 0 {
		t.Errorf("Expected length 0 after clear, got %d", c.Len())
	}

	_, found := c.Get("key1")
	if found {
		t.Error("Expected no keys after clear")
	}
}

func TestCacheGetOrSet(t *testing.T) {
	c := New[string, int](time.Minute)

	callCount := 0
	loader := func() (int, error) {
		callCount++
		return 42, nil
	}

	// First call should invoke loader
	value, err := c.GetOrSet("key1", loader)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if value != 42 {
		t.Errorf("Expected 42, got %d", value)
	}
	if callCount != 1 {
		t.Errorf("Expected loader to be called once, was called %d times", callCount)
	}

	// Second call should use cached value
	value, err = c.GetOrSet("key1", loader)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if value != 42 {
		t.Errorf("Expected 42, got %d", value)
	}
	if callCount != 1 {
		t.Errorf("Expected loader not to be called again, was called %d times", callCount)
	}
}

func TestCacheInvalidatePattern(t *testing.T) {
	c := New[string, string](time.Minute)

	c.Set("repo1:key1", "value1")
	c.Set("repo1:key2", "value2")
	c.Set("repo2:key1", "value3")
	c.Set("repo2:key2", "value4")

	// Invalidate all repo1 keys
	c.InvalidatePattern(func(key string) bool {
		return len(key) >= 5 && key[:5] == "repo1"
	})

	// repo1 keys should be gone
	_, found := c.Get("repo1:key1")
	if found {
		t.Error("Expected repo1:key1 to be invalidated")
	}
	_, found = c.Get("repo1:key2")
	if found {
		t.Error("Expected repo1:key2 to be invalidated")
	}

	// repo2 keys should still exist
	value, found := c.Get("repo2:key1")
	if !found {
		t.Error("Expected repo2:key1 to still exist")
	}
	if value != "value3" {
		t.Errorf("Expected value3, got %s", value)
	}
}

func TestCacheKeys(t *testing.T) {
	c := New[string, string](time.Minute)

	c.Set("key1", "value1")
	c.Set("key2", "value2")
	c.Set("key3", "value3")

	keys := c.Keys()
	if len(keys) != 3 {
		t.Errorf("Expected 3 keys, got %d", len(keys))
	}

	// Verify all keys are present
	keyMap := make(map[string]bool)
	for _, key := range keys {
		keyMap[key] = true
	}

	if !keyMap["key1"] || !keyMap["key2"] || !keyMap["key3"] {
		t.Error("Not all expected keys are present")
	}
}

func TestCacheStats(t *testing.T) {
	c := New[string, string](100 * time.Millisecond)

	c.Set("key1", "value1")
	c.Set("key2", "value2")

	stats := c.GetStats()
	if stats.TotalItems != 2 {
		t.Errorf("Expected 2 total items, got %d", stats.TotalItems)
	}
	if stats.ExpiredItems != 0 {
		t.Errorf("Expected 0 expired items, got %d", stats.ExpiredItems)
	}

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	stats = c.GetStats()
	if stats.TotalItems != 2 {
		t.Errorf("Expected 2 total items, got %d", stats.TotalItems)
	}
	if stats.ExpiredItems != 2 {
		t.Errorf("Expected 2 expired items, got %d", stats.ExpiredItems)
	}
}

func TestCacheTTLGetSet(t *testing.T) {
	c := New[string, string](time.Minute)

	ttl := c.GetTTL()
	if ttl != time.Minute {
		t.Errorf("Expected TTL of 1 minute, got %v", ttl)
	}

	c.SetTTL(time.Hour)

	ttl = c.GetTTL()
	if ttl != time.Hour {
		t.Errorf("Expected TTL of 1 hour, got %v", ttl)
	}
}

func TestCacheConcurrency(t *testing.T) {
	c := New[int, int](time.Minute)

	var wg sync.WaitGroup
	numGoroutines := 100
	operationsPerGoroutine := 100

	// Launch goroutines that concurrently read and write
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			for j := 0; j < operationsPerGoroutine; j++ {
				key := (id * operationsPerGoroutine) + j

				// Set
				c.Set(key, key*2)

				// Get
				value, found := c.Get(key)
				if found && value != key*2 {
					t.Errorf("Expected %d, got %d", key*2, value)
				}

				// Delete every other key
				if j%2 == 0 {
					c.Delete(key)
				}
			}
		}(i)
	}

	wg.Wait()
}

func TestCacheRemoveExpired(t *testing.T) {
	c := New[string, string](100 * time.Millisecond)

	c.Set("key1", "value1")
	c.Set("key2", "value2")
	c.Set("key3", "value3")

	if c.Len() != 3 {
		t.Errorf("Expected length 3, got %d", c.Len())
	}

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Manually trigger cleanup
	c.removeExpired()

	// All items should be removed
	if c.Len() != 0 {
		t.Errorf("Expected length 0 after cleanup, got %d", c.Len())
	}
}

func TestCacheGetOrSetError(t *testing.T) {
	c := New[string, int](time.Minute)

	loader := func() (int, error) {
		return 0, &testError{"loader error"}
	}

	_, err := c.GetOrSet("key1", loader)
	if err == nil {
		t.Error("Expected error from loader")
	}

	// Key should not be in cache after error
	_, found := c.Get("key1")
	if found {
		t.Error("Expected key not to be cached after loader error")
	}
}

// testError is a simple error type for testing
type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}

// Benchmark tests
func BenchmarkCacheSet(b *testing.B) {
	c := New[int, int](time.Minute)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Set(i, i*2)
	}
}

func BenchmarkCacheGet(b *testing.B) {
	c := New[int, int](time.Minute)

	// Pre-populate cache
	for i := 0; i < 1000; i++ {
		c.Set(i, i*2)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Get(i % 1000)
	}
}

func BenchmarkCacheGetOrSet(b *testing.B) {
	c := New[int, int](time.Minute)

	loader := func() (int, error) {
		return 42, nil
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.GetOrSet(i%100, loader)
	}
}

func BenchmarkCacheConcurrentReads(b *testing.B) {
	c := New[int, int](time.Minute)

	// Pre-populate cache
	for i := 0; i < 1000; i++ {
		c.Set(i, i*2)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			c.Get(i % 1000)
			i++
		}
	})
}

func BenchmarkCacheConcurrentWrites(b *testing.B) {
	c := New[int, int](time.Minute)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			c.Set(i, i*2)
			i++
		}
	})
}
