package steam

import (
	"sync"
	"time"
)

const (
	appDetailsCacheTTL = 10 * time.Minute
	storeItemCacheTTL  = 10 * time.Minute
	profileCacheTTL    = 10 * time.Minute
	wishlistCacheTTL   = 5 * time.Minute
)

type cacheEntry[T any] struct {
	value     T
	expiresAt time.Time
}

// Cache is a first-class in-memory cache shared by one or more Clients.
// All accessor methods are goroutine-safe.
type Cache struct {
	mu         sync.Mutex
	appDetails map[string]cacheEntry[*AppDetails]
	storeItem  map[string]cacheEntry[*StoreItem]
	profiles   map[string]cacheEntry[*UserProfile]
	wishlists  map[string]cacheEntry[[]wishlistRawItem]
}

// NewCache returns a fresh empty cache. Tests should construct their own
// cache to avoid bleed; production code shares one cache via the package-level
// DefaultCache so regional clients can hit each other's app-detail entries.
func NewCache() *Cache {
	return &Cache{
		appDetails: map[string]cacheEntry[*AppDetails]{},
		storeItem:  map[string]cacheEntry[*StoreItem]{},
		profiles:   map[string]cacheEntry[*UserProfile]{},
		wishlists:  map[string]cacheEntry[[]wishlistRawItem]{},
	}
}

// DefaultCache is the process-wide cache used when callers don't supply one.
// Multiple Clients (e.g. one per --cc in price comparison) share it.
var DefaultCache = NewCache()

func (c *Cache) GetAppDetails(key string) (*AppDetails, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return cacheGet(c.appDetails, key)
}

func (c *Cache) SetAppDetails(key string, value *AppDetails) {
	c.mu.Lock()
	defer c.mu.Unlock()
	cacheSet(c.appDetails, key, value, appDetailsCacheTTL)
}

func (c *Cache) GetStoreItem(key string) (*StoreItem, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return cacheGet(c.storeItem, key)
}

func (c *Cache) SetStoreItem(key string, value *StoreItem) {
	c.mu.Lock()
	defer c.mu.Unlock()
	cacheSet(c.storeItem, key, value, storeItemCacheTTL)
}

func (c *Cache) GetProfile(key string) (*UserProfile, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return cacheGet(c.profiles, key)
}

func (c *Cache) SetProfile(key string, value *UserProfile) {
	c.mu.Lock()
	defer c.mu.Unlock()
	cacheSet(c.profiles, key, value, profileCacheTTL)
}

func (c *Cache) GetWishlist(key string) ([]wishlistRawItem, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return cacheGet(c.wishlists, key)
}

func (c *Cache) SetWishlist(key string, value []wishlistRawItem) {
	c.mu.Lock()
	defer c.mu.Unlock()
	cacheSet(c.wishlists, key, value, wishlistCacheTTL)
}

func cacheGet[T any](store map[string]cacheEntry[T], key string) (T, bool) {
	entry, ok := store[key]
	if !ok || time.Now().After(entry.expiresAt) {
		var zero T
		return zero, false
	}
	return entry.value, true
}

func cacheSet[T any](store map[string]cacheEntry[T], key string, value T, ttl time.Duration) {
	store[key] = cacheEntry[T]{
		value:     value,
		expiresAt: time.Now().Add(ttl),
	}
}
