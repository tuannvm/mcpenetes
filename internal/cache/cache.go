package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log" // Import standard log package
	"net/url"
	"os"
	"path/filepath"
	"time"
	// Keep internal log if needed elsewhere, otherwise remove
	// internalLog "github.com/tuannvm/mcpenetes/internal/log"
)

const (
	// cacheTTL defines how long cache entries are considered valid.
	cacheTTL = 1 * time.Hour
)

// CacheEntry represents the structure of the data stored in a cache file.
type CacheEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Versions  []string  `json:"versions"`
}

// cacheDirPath stores the path to the cache directory. Initialized by init().
var cacheDirPath string

// init ensures the cache directory exists.
func init() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("failed to get user home directory: %v", err) // Use standard log.Fatalf
	}
	cacheDirPath = filepath.Join(homeDir, ".config/mcpetes/cache")
	if err := os.MkdirAll(cacheDirPath, 0755); err != nil {
		log.Fatalf("failed to create cache directory '%s': %v", cacheDirPath, err) // Use standard log.Fatalf
	}
}

// getCachePath generates a unique and safe file path for a given registry URL.
// It's defined as a variable to allow mocking in tests.
var getCachePath = func(registryURL string) (string, error) {
	// Basic URL validation
	parsedURL, err := url.Parse(registryURL)
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		return "", fmt.Errorf("invalid registry URL for cache key: %w", err)
	}

	// Create a safe filename from the URL (e.g., hash or escape)
	// Using a hash of the URL ensures it's unique and avoids filesystem issues
	hasher := sha256.New()
	hasher.Write([]byte(registryURL))
	cacheKey := hex.EncodeToString(hasher.Sum(nil))
	cacheFileName := cacheKey + ".json"

	return filepath.Join(cacheDirPath, cacheFileName), nil
}

// ReadCache reads the cached versions for a registry URL if the cache is valid.
// Returns the list of versions, a boolean indicating if it was a cache miss (or expired/invalid), and any error encountered.
func ReadCache(registryURL string) (versions []string, cacheMiss bool, err error) {
	cachePath, err := getCachePath(registryURL)
	if err != nil {
		return nil, true, fmt.Errorf("failed to get cache path: %w", err)
	}

	data, err := os.ReadFile(cachePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, true, nil // Cache miss, not an error
		}
		return nil, false, fmt.Errorf("failed to read cache file '%s': %w", cachePath, err)
	}

	var entry CacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		// Treat invalid cache data as a cache miss, maybe log it
		fmt.Fprintf(os.Stderr, "Warning: Failed to parse cache file '%s', ignoring: %v\n", cachePath, err)
		return nil, true, nil
	}

	// Check if cache entry is expired
	if time.Since(entry.Timestamp) > cacheTTL {
		return nil, true, nil // Cache expired, treat as miss
	}

	return entry.Versions, false, nil // Cache hit and valid
}

// WriteCache writes the fetched versions to the cache file for a registry URL.
func WriteCache(registryURL string, versions []string) error {
	cachePath, err := getCachePath(registryURL)
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

	if err := os.WriteFile(cachePath, data, 0600); err != nil {
		return fmt.Errorf("failed to write cache file '%s': %w", cachePath, err)
	}

	return nil
}
