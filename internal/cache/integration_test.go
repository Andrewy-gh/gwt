package cache

import (
	"sync"
	"testing"
	"time"
)

// TestIntegration_CacheLifecycle tests the complete cache lifecycle
func TestIntegration_CacheLifecycle(t *testing.T) {
	cache := New[string, string](time.Minute)

	// 1. Set initial value
	cache.Set("key1", "value1")

	// 2. Get value (cache hit)
	val, found := cache.Get("key1")
	if !found {
		t.Error("Expected cache hit, got miss")
	}
	if val != "value1" {
		t.Errorf("Expected value1, got %s", val)
	}

	// 3. Update value
	cache.Set("key1", "value2")
	val, found = cache.Get("key1")
	if !found || val != "value2" {
		t.Error("Cache update failed")
	}

	// 4. Delete value
	cache.Delete("key1")
	_, found = cache.Get("key1")
	if found {
		t.Error("Value should be deleted")
	}

	// 5. GetOrSet with loader
	val, err := cache.GetOrSet("key2", func() (string, error) {
		return "loaded", nil
	})
	if err != nil || val != "loaded" {
		t.Error("GetOrSet failed")
	}

	// 6. Verify GetOrSet cached the value
	val, found = cache.Get("key2")
	if !found || val != "loaded" {
		t.Error("GetOrSet should have cached value")
	}

	// 7. Clear all
	cache.Clear()
	_, found = cache.Get("key2")
	if found {
		t.Error("Cache should be empty after Clear")
	}
}

// TestIntegration_TTLExpiration tests cache expiration behavior
func TestIntegration_TTLExpiration(t *testing.T) {
	shortTTL := 100 * time.Millisecond
	cache := New[string, string](shortTTL)

	// Set a value
	cache.Set("key1", "value1")

	// Immediate read should succeed
	val, found := cache.Get("key1")
	if !found || val != "value1" {
		t.Error("Immediate read should succeed")
	}

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Read should fail after TTL
	_, found = cache.Get("key1")
	if found {
		t.Error("Value should be expired")
	}

	// Test custom TTL
	cache.SetWithTTL("key2", "value2", 200*time.Millisecond)

	// Should exist initially
	_, found = cache.Get("key2")
	if !found {
		t.Error("Value should exist")
	}

	// Wait for custom TTL
	time.Sleep(250 * time.Millisecond)

	// Should be expired
	_, found = cache.Get("key2")
	if found {
		t.Error("Custom TTL value should be expired")
	}
}

// TestIntegration_ConcurrentAccess tests concurrent cache operations
func TestIntegration_ConcurrentAccess(t *testing.T) {
	cache := New[string, int](time.Minute)
	numGoroutines := 50
	opsPerGoroutine := 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Concurrent writes
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < opsPerGoroutine; j++ {
				key := string(rune('a' + (id % 26)))
				cache.Set(key, id*opsPerGoroutine+j)
			}
		}(i)
	}

	wg.Wait()

	// Verify cache still works
	cache.Set("test", 42)
	val, found := cache.Get("test")
	if !found || val != 42 {
		t.Error("Cache corrupted after concurrent access")
	}

	// Concurrent reads
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < opsPerGoroutine; j++ {
				cache.Get("test")
			}
		}()
	}

	wg.Wait()

	// Verify value unchanged
	val, found = cache.Get("test")
	if !found || val != 42 {
		t.Error("Concurrent reads corrupted cache")
	}
}

// TestIntegration_PatternInvalidation tests pattern-based cache invalidation
func TestIntegration_PatternInvalidation(t *testing.T) {
	cache := New[string, string](time.Minute)

	// Set up test data with different patterns
	cache.Set("repo1:worktrees", "data1")
	cache.Set("repo1:branches:local", "data2")
	cache.Set("repo1:branches:all", "data3")
	cache.Set("repo2:worktrees", "data4")
	cache.Set("repo2:branches:local", "data5")

	// Invalidate all repo1 branch caches
	cache.InvalidatePattern(func(key string) bool {
		return key == "repo1:branches:local" || key == "repo1:branches:all"
	})

	// Verify repo1 branches are gone
	if _, found := cache.Get("repo1:branches:local"); found {
		t.Error("repo1:branches:local should be invalidated")
	}
	if _, found := cache.Get("repo1:branches:all"); found {
		t.Error("repo1:branches:all should be invalidated")
	}

	// Verify repo1 worktrees still exists
	if _, found := cache.Get("repo1:worktrees"); !found {
		t.Error("repo1:worktrees should still exist")
	}

	// Verify repo2 caches still exist
	if _, found := cache.Get("repo2:worktrees"); !found {
		t.Error("repo2:worktrees should still exist")
	}
	if _, found := cache.Get("repo2:branches:local"); !found {
		t.Error("repo2:branches:local should still exist")
	}
}

// TestIntegration_GetOrSetConcurrent tests concurrent GetOrSet with loader
func TestIntegration_GetOrSetConcurrent(t *testing.T) {
	cache := New[string, int](time.Minute)

	callCount := 0
	var mu sync.Mutex

	loader := func() (int, error) {
		mu.Lock()
		callCount++
		mu.Unlock()
		time.Sleep(50 * time.Millisecond) // Simulate slow operation
		return 42, nil
	}

	// Multiple goroutines call GetOrSet concurrently
	numGoroutines := 10
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	results := make([]int, numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			val, err := cache.GetOrSet("shared-key", loader)
			if err != nil {
				t.Errorf("GetOrSet failed: %v", err)
				return
			}
			results[id] = val
		}(i)
	}

	wg.Wait()

	// All goroutines should get the same value
	for i, val := range results {
		if val != 42 {
			t.Errorf("Goroutine %d got wrong value: %d", i, val)
		}
	}

	// Loader might be called multiple times due to race conditions,
	// but that's acceptable. The important thing is all get correct value.
	t.Logf("Loader called %d times (expected: 1-few due to races)", callCount)
}

// TestIntegration_CacheStats tests cache statistics
func TestIntegration_CacheStats(t *testing.T) {
	cache := New[string, string](time.Minute)

	// Initially empty
	stats := cache.GetStats()
	if stats.TotalItems != 0 || stats.ExpiredItems != 0 {
		t.Error("Cache should be empty initially")
	}

	// Add some items
	cache.Set("key1", "value1")
	cache.Set("key2", "value2")
	cache.Set("key3", "value3")

	stats = cache.GetStats()
	if stats.TotalItems != 3 {
		t.Errorf("Expected 3 items, got %d", stats.TotalItems)
	}

	// Add expired item
	cache.SetWithTTL("expired", "value", 1*time.Millisecond)
	time.Sleep(10 * time.Millisecond)

	stats = cache.GetStats()
	if stats.TotalItems != 4 {
		t.Errorf("Expected 4 total items, got %d", stats.TotalItems)
	}
	if stats.ExpiredItems != 1 {
		t.Errorf("Expected 1 expired item, got %d", stats.ExpiredItems)
	}
}

// TestIntegration_CacheTTLUpdate tests updating cache TTL
func TestIntegration_CacheTTLUpdate(t *testing.T) {
	cache := New[string, string](time.Hour)

	// Add an item with long TTL
	cache.Set("key1", "value1")

	// Update TTL to be short
	cache.SetTTL(50 * time.Millisecond)

	// Add another item (uses new TTL)
	cache.Set("key2", "value2")

	// Wait for new TTL to expire
	time.Sleep(100 * time.Millisecond)

	// key1 still exists (has old TTL)
	if _, found := cache.Get("key1"); !found {
		t.Error("key1 should still exist with original TTL")
	}

	// key2 should be expired (has new TTL)
	if _, found := cache.Get("key2"); found {
		t.Error("key2 should be expired with new TTL")
	}
}

// TestIntegration_KeysListing tests listing all cache keys
func TestIntegration_KeysListing(t *testing.T) {
	cache := New[string, string](time.Minute)

	// Add several items
	expected := map[string]bool{
		"key1": true,
		"key2": true,
		"key3": true,
	}

	for key := range expected {
		cache.Set(key, "value")
	}

	// Get all keys
	keys := cache.Keys()

	if len(keys) != len(expected) {
		t.Errorf("Expected %d keys, got %d", len(expected), len(keys))
	}

	// Verify all expected keys are present
	for _, key := range keys {
		if !expected[key] {
			t.Errorf("Unexpected key: %s", key)
		}
		delete(expected, key)
	}

	if len(expected) > 0 {
		t.Errorf("Missing keys: %v", expected)
	}
}

// TestIntegration_ComplexDataTypes tests cache with complex types
func TestIntegration_ComplexDataTypes(t *testing.T) {
	type ComplexStruct struct {
		ID       int
		Name     string
		Items    []string
		Metadata map[string]interface{}
	}

	cache := New[string, ComplexStruct](time.Minute)

	original := ComplexStruct{
		ID:    123,
		Name:  "test",
		Items: []string{"a", "b", "c"},
		Metadata: map[string]interface{}{
			"key1": "value1",
			"key2": 42,
		},
	}

	cache.Set("complex", original)

	retrieved, found := cache.Get("complex")
	if !found {
		t.Fatal("Complex struct not found in cache")
	}

	// Verify all fields
	if retrieved.ID != original.ID {
		t.Error("ID mismatch")
	}
	if retrieved.Name != original.Name {
		t.Error("Name mismatch")
	}
	if len(retrieved.Items) != len(original.Items) {
		t.Error("Items length mismatch")
	}
	if len(retrieved.Metadata) != len(original.Metadata) {
		t.Error("Metadata length mismatch")
	}
}
