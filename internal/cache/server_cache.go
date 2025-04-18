package cache

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// ServerInfo represents information about an MCP server to be cached
type ServerInfo struct {
	Name          string `json:"name"`
	Description   string `json:"description"`
	RepositoryURL string `json:"repositoryUrl"`
}

// ServerCacheEntry represents the structure of server data stored in a cache file
type ServerCacheEntry struct {
	Timestamp time.Time    `json:"timestamp"`
	Servers   []ServerInfo `json:"servers"`
}

// ReadServerCache reads the cached server information for a registry URL if the cache is valid.
// Returns the list of servers, a boolean indicating if it was a cache miss (or expired/invalid), and any error encountered.
func ReadServerCache(registryURL string) (servers []ServerInfo, cacheMiss bool, err error) {
	cachePath, err := getCachePath(registryURL + "-servers") // Append suffix to differentiate from version cache
	if err != nil {
		return nil, false, fmt.Errorf("failed to get server cache path: %w", err)
	}

	data, err := os.ReadFile(cachePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, true, nil // Cache miss, not an error
		}
		return nil, false, fmt.Errorf("failed to read server cache file '%s': %w", cachePath, err)
	}

	var entry ServerCacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		// Treat invalid cache data as a cache miss
		return nil, true, nil
	}

	// Check if cache entry is expired
	if time.Since(entry.Timestamp) > cacheTTL {
		return nil, true, nil // Cache expired, treat as miss
	}

	return entry.Servers, false, nil // Cache hit and valid
}

// WriteServerCache writes the fetched server information to the cache file for a registry URL.
func WriteServerCache(registryURL string, servers []ServerInfo) error {
	cachePath, err := getCachePath(registryURL + "-servers") // Append suffix to differentiate
	if err != nil {
		return fmt.Errorf("failed to get server cache path for writing: %w", err)
	}

	entry := ServerCacheEntry{
		Timestamp: time.Now(),
		Servers:   servers,
	}

	data, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal server cache entry to JSON: %w", err)
	}

	if err := os.WriteFile(cachePath, data, 0600); err != nil {
		return fmt.Errorf("failed to write server cache file '%s': %w", cachePath, err)
	}

	return nil
}

// ClearServerCache clears the server cache for the given registry URL.
func ClearServerCache(registryURL string) error {
	cachePath, err := getCachePath(registryURL + "-servers")
	if err != nil {
		return fmt.Errorf("failed to get server cache path for clearing: %w", err)
	}

	// Remove the cache file if it exists
	if err := os.Remove(cachePath); err != nil {
		if os.IsNotExist(err) {
			return nil // File doesn't exist, consider it a success
		}
		return fmt.Errorf("failed to remove server cache file '%s': %w", cachePath, err)
	}

	return nil
}
