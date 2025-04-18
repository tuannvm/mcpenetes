package cache

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	cacheDirName = ".config/mcpetes/cache"
	cacheTTL     = 1 * time.Hour // How long cache entries are considered valid
)

// CacheEntry represents the data stored in a cache file.
type CacheEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Versions  []string  `json:"versions"`
}

// getCachePath generates a unique cache file path for a given URL.
func getCachePath(registryURL string) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}
	cacheDirPath := filepath.Join(homeDir, cacheDirName)

	// Create a safe filename from the URL (e.g., hash or escape)
	// Using URL hostname + path escaping is reasonably safe and readable
	u, err := url.Parse(registryURL)
	if err != nil {
		return "", fmt.Errorf("invalid registry URL for cache key: %w", err)
	}
	cacheKey := fmt.Sprintf("%s%s", u.Host, strings.ReplaceAll(u.Path, "/", "_"))
	cacheKey = url.PathEscape(cacheKey) // Ensure it's filename safe
	cacheFileName := cacheKey + ".json"

	return filepath.Join(cacheDirPath, cacheFileName), nil
}

// ReadCache attempts to read a valid cache entry for a given URL.
func ReadCache(registryURL string) (*CacheEntry, error) {
	cacheFilePath, err := getCachePath(registryURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get cache path: %w", err)
	}

	data, err := os.ReadFile(cacheFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // Cache miss, not an error
		}
		return nil, fmt.Errorf("failed to read cache file '%s': %w", cacheFilePath, err)
	}

	var entry CacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		// Treat invalid cache data as a cache miss, maybe log it
		fmt.Fprintf(os.Stderr, "Warning: Failed to parse cache file '%s', ignoring: %v\n", cacheFilePath, err)
		return nil, nil
	}

	// Check if cache entry is expired
	if time.Since(entry.Timestamp) > cacheTTL {
		return nil, nil // Cache expired, treat as miss
	}

	return &entry, nil // Cache hit and valid
}

// WriteCache writes fetched versions to the cache file for a given URL.
func WriteCache(registryURL string, versions []string) error {
	cacheFilePath, err := getCachePath(registryURL)
	if err != nil {
		return fmt.Errorf("failed to get cache path for writing: %w", err)
	}

	entry := CacheEntry{
		Timestamp: time.Now(),
		Versions:  versions,
	}

	data, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal cache entry to JSON: %w", err)
	}

	// Ensure cache directory exists
	cacheDirPath := filepath.Dir(cacheFilePath)
	if err := os.MkdirAll(cacheDirPath, 0750); err != nil {
		return fmt.Errorf("failed to create cache directory '%s': %w", cacheDirPath, err)
	}

	if err := os.WriteFile(cacheFilePath, data, 0600); err != nil {
		return fmt.Errorf("failed to write cache file '%s': %w", cacheFilePath, err)
	}

	return nil
}
