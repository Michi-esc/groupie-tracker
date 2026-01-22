package models

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// GeocodeCache stores cached geocoded locations
type GeocodeCache struct {
	Locations map[string]LocationCoords `json:"locations"`
	mu        sync.RWMutex
}

var (
	geocodeCache     = &GeocodeCache{Locations: make(map[string]LocationCoords)}
	geocodeCachePath = ""
	cacheSaveTimer   *time.Timer
	saveCacheMu      sync.Mutex
)

// InitGeocodeCache initializes the geocode cache from disk
func InitGeocodeCache() error {
	// use a cache file in the user's home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}
	geocodeCachePath = filepath.Join(homeDir, ".groupie-tracker-geocache.json")

	// load existing cache from disk
	if data, err := os.ReadFile(geocodeCachePath); err == nil {
		if err := json.Unmarshal(data, geocodeCache); err == nil {
			log.Printf("[✓ GEOCACHE] Loaded %d entries from cache\n", len(geocodeCache.Locations))
			return nil
		}
	}
	log.Printf("[✓ GEOCACHE] Starting fresh cache at %s\n", geocodeCachePath)
	return nil
}

// GetCachedCoords returns cached coordinates if available
func GetCachedCoords(location string) *LocationCoords {
	geocodeCache.mu.Lock()
	defer geocodeCache.mu.Unlock()

	if coords, exists := geocodeCache.Locations[location]; exists {
		log.Printf("[CACHE HIT] %s\n", location)
		return &coords
	}
	return nil
}

// CacheCoords stores coordinates in memory and persists to disk
func CacheCoords(location string, coords *LocationCoords) {
	if coords == nil {
		return
	}

	geocodeCache.mu.Lock()
	geocodeCache.Locations[location] = *coords
	geocodeCache.mu.Unlock()

	// Debounce cache saves - wait 2 seconds then save once
	saveCacheMu.Lock()
	if cacheSaveTimer != nil {
		cacheSaveTimer.Stop()
	}
	cacheSaveTimer = time.AfterFunc(2*time.Second, saveCacheNow)
	saveCacheMu.Unlock()
}

// saveCacheNow synchronously saves the cache to disk
func saveCacheNow() {
	geocodeCache.mu.Lock()
	defer geocodeCache.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Create a channel to signal completion
	done := make(chan error, 1)
	go func() {
		data, err := json.MarshalIndent(geocodeCache, "", "  ")
		if err != nil {
			done <- err
			return
		}
		done <- os.WriteFile(geocodeCachePath, data, 0644)
	}()

	// Wait for either completion or timeout
	select {
	case err := <-done:
		if err != nil {
			log.Printf("[WARN] Failed to save geocode cache: %v\n", err)
		}
	case <-ctx.Done():
		log.Printf("[WARN] Timeout saving geocode cache\n")
	}
}

// GetCacheSize returns the number of cached entries
func GetCacheSize() int {
	geocodeCache.mu.RLock()
	defer geocodeCache.mu.Unlock()
	return len(geocodeCache.Locations)
}

// ClearCache clears all cached entries
func ClearCache() error {
	geocodeCache.mu.Lock()
	geocodeCache.Locations = make(map[string]LocationCoords)
	geocodeCache.mu.Unlock()

	return os.Remove(geocodeCachePath)
}
