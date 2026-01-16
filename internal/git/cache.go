package git

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/Andrewy-gh/gwt/internal/cache"
)

var (
	// Global caches with their respective TTLs
	worktreeListCache *cache.Cache[string, []Worktree]
	branchListCache   *cache.Cache[string, []Branch]
	statusCache       *cache.Cache[string, *WorktreeStatus]

	// Cache initialization
	cacheOnce    sync.Once
	cacheEnabled = true
	cacheMu      sync.RWMutex

	// Default TTLs
	defaultWorktreeTTL = time.Minute
	defaultBranchTTL   = 5 * time.Minute
	defaultStatusTTL   = 30 * time.Second
)

// initCaches initializes all git caches
func initCaches() {
	cacheOnce.Do(func() {
		worktreeListCache = cache.New[string, []Worktree](defaultWorktreeTTL)
		branchListCache = cache.New[string, []Branch](defaultBranchTTL)
		statusCache = cache.New[string, *WorktreeStatus](defaultStatusTTL)
	})
}

// SetCacheEnabled enables or disables caching globally
// Useful for testing and troubleshooting
func SetCacheEnabled(enabled bool) {
	cacheMu.Lock()
	defer cacheMu.Unlock()
	cacheEnabled = enabled

	// Clear all caches when disabling
	if !enabled {
		initCaches()
		worktreeListCache.Clear()
		branchListCache.Clear()
		statusCache.Clear()
	}
}

// IsCacheEnabled returns whether caching is currently enabled
func IsCacheEnabled() bool {
	cacheMu.RLock()
	defer cacheMu.RUnlock()
	return cacheEnabled
}

// SetCacheTTLs updates the TTL values for all caches
func SetCacheTTLs(worktreeTTL, branchTTL, statusTTL time.Duration) {
	initCaches()

	if worktreeTTL > 0 {
		worktreeListCache.SetTTL(worktreeTTL)
	}
	if branchTTL > 0 {
		branchListCache.SetTTL(branchTTL)
	}
	if statusTTL > 0 {
		statusCache.SetTTL(statusTTL)
	}
}

// ListWorktreesCached returns worktrees with caching support
func ListWorktreesCached(repoPath string, bypassCache bool) ([]Worktree, error) {
	if !IsCacheEnabled() || bypassCache {
		return ListWorktrees(repoPath)
	}

	initCaches()

	cacheKey := fmt.Sprintf("%s:worktrees", repoPath)
	return worktreeListCache.GetOrSet(cacheKey, func() ([]Worktree, error) {
		return ListWorktrees(repoPath)
	})
}

// ListLocalBranchesCached returns local branches with caching support
func ListLocalBranchesCached(repoPath string, bypassCache bool) ([]Branch, error) {
	if !IsCacheEnabled() || bypassCache {
		return ListLocalBranches(repoPath)
	}

	initCaches()

	cacheKey := fmt.Sprintf("%s:branches:local", repoPath)
	return branchListCache.GetOrSet(cacheKey, func() ([]Branch, error) {
		return ListLocalBranches(repoPath)
	})
}

// ListAllBranchesCached returns all branches with caching support
func ListAllBranchesCached(repoPath string, bypassCache bool) ([]Branch, error) {
	if !IsCacheEnabled() || bypassCache {
		return ListAllBranches(repoPath)
	}

	initCaches()

	cacheKey := fmt.Sprintf("%s:branches:all", repoPath)
	return branchListCache.GetOrSet(cacheKey, func() ([]Branch, error) {
		return ListAllBranches(repoPath)
	})
}

// GetWorktreeStatusCached returns worktree status with caching support
func GetWorktreeStatusCached(worktreePath string, bypassCache bool) (*WorktreeStatus, error) {
	if !IsCacheEnabled() || bypassCache {
		return GetWorktreeStatus(worktreePath)
	}

	initCaches()

	cacheKey := fmt.Sprintf("%s:status", worktreePath)
	return statusCache.GetOrSet(cacheKey, func() (*WorktreeStatus, error) {
		return GetWorktreeStatus(worktreePath)
	})
}

// GetWorktreeStatusBatchCached fetches status for multiple worktrees with caching
// Falls back to GetWorktreeStatusBatch for cache misses, fetches them in parallel
func GetWorktreeStatusBatchCached(ctx context.Context, paths []string, maxWorkers int, bypassCache bool) (map[string]*WorktreeStatus, []error) {
	if !IsCacheEnabled() || bypassCache {
		return GetWorktreeStatusBatch(ctx, paths, maxWorkers)
	}

	initCaches()

	result := make(map[string]*WorktreeStatus)
	var cacheMisses []string
	var mu sync.Mutex

	// Check cache for each path
	for _, path := range paths {
		cacheKey := fmt.Sprintf("%s:status", path)
		if status, found := statusCache.Get(cacheKey); found {
			mu.Lock()
			result[path] = status
			mu.Unlock()
		} else {
			cacheMisses = append(cacheMisses, path)
		}
	}

	// If all results were cached, return early
	if len(cacheMisses) == 0 {
		return result, nil
	}

	// Fetch cache misses in parallel
	batchResult, errors := GetWorktreeStatusBatch(ctx, cacheMisses, maxWorkers)

	// Store results in cache and add to result map
	for path, status := range batchResult {
		cacheKey := fmt.Sprintf("%s:status", path)
		statusCache.Set(cacheKey, status)

		mu.Lock()
		result[path] = status
		mu.Unlock()
	}

	return result, errors
}

// InvalidateWorktreeCache invalidates all worktree-related cache entries for a repository
func InvalidateWorktreeCache(repoPath string) {
	if !IsCacheEnabled() {
		return
	}

	initCaches()

	// Invalidate worktree list cache
	cacheKey := fmt.Sprintf("%s:worktrees", repoPath)
	worktreeListCache.Delete(cacheKey)

	// Invalidate all status caches for worktrees in this repo
	// Pattern: any cache key that starts with a path under this repo
	statusCache.InvalidatePattern(func(key string) bool {
		// Extract path from cache key (format: "path:status")
		if !strings.HasSuffix(key, ":status") {
			return false
		}
		path := strings.TrimSuffix(key, ":status")

		// Check if this worktree path is under the repo path
		// This is a simple string prefix check
		return strings.HasPrefix(path, repoPath)
	})
}

// InvalidateBranchCache invalidates all branch-related cache entries for a repository
func InvalidateBranchCache(repoPath string) {
	if !IsCacheEnabled() {
		return
	}

	initCaches()

	// Invalidate all branch list caches
	branchListCache.InvalidatePattern(func(key string) bool {
		// Pattern: "repoPath:branches:*"
		prefix := fmt.Sprintf("%s:branches:", repoPath)
		return strings.HasPrefix(key, prefix)
	})
}

// InvalidateStatusCache invalidates status cache for a specific worktree
func InvalidateStatusCache(worktreePath string) {
	if !IsCacheEnabled() {
		return
	}

	initCaches()

	cacheKey := fmt.Sprintf("%s:status", worktreePath)
	statusCache.Delete(cacheKey)
}

// ClearAllCaches clears all git-related caches
func ClearAllCaches() {
	if !IsCacheEnabled() {
		return
	}

	initCaches()

	worktreeListCache.Clear()
	branchListCache.Clear()
	statusCache.Clear()
}

// GetCacheStats returns statistics for all caches
type CacheStats struct {
	WorktreeList cache.Stats
	BranchList   cache.Stats
	Status       cache.Stats
}

// GetCacheStats returns current cache statistics
func GetCacheStats() CacheStats {
	if !IsCacheEnabled() {
		return CacheStats{}
	}

	initCaches()

	return CacheStats{
		WorktreeList: worktreeListCache.GetStats(),
		BranchList:   branchListCache.GetStats(),
		Status:       statusCache.GetStats(),
	}
}
