package cache

import (
	"path/filepath"
	"sync"
	"time"

	"github.com/ianzepp/monk-api-fuse/pkg/monkapi"
)

// MetadataCache caches file and directory metadata to reduce API calls
type MetadataCache struct {
	mu      sync.RWMutex
	entries map[string]*CacheEntry
	ttl     time.Duration
}

// CacheEntry represents a cached metadata entry
type CacheEntry struct {
	data      *monkapi.StatResponse
	timestamp time.Time
}

// NewMetadataCache creates a new metadata cache with the specified TTL
func NewMetadataCache(ttl time.Duration) *MetadataCache {
	return &MetadataCache{
		entries: make(map[string]*CacheEntry),
		ttl:     ttl,
	}
}

// Get retrieves metadata from cache if available and not expired
func (c *MetadataCache) Get(path string) *monkapi.StatResponse {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.entries[path]
	if !ok {
		return nil
	}

	// Check TTL
	if time.Since(entry.timestamp) > c.ttl {
		return nil
	}

	return entry.data
}

// Set stores metadata in cache
func (c *MetadataCache) Set(path string, data *monkapi.StatResponse) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries[path] = &CacheEntry{
		data:      data,
		timestamp: time.Now(),
	}
}

// Invalidate removes a path and its parent directories from cache
func (c *MetadataCache) Invalidate(path string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.entries, path)

	// Invalidate parent directories
	for parent := filepath.Dir(path); parent != "/" && parent != "."; parent = filepath.Dir(parent) {
		delete(c.entries, parent)
	}
}

// Clear removes all entries from cache
func (c *MetadataCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries = make(map[string]*CacheEntry)
}
