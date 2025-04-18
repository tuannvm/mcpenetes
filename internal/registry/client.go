package registry

import (
	"encoding/json"
	"fmt" // Re-add fmt import for Errorf
	"net/http"
	"time"

	"github.com/tuannvm/mcpenetes/internal/cache"
	"github.com/tuannvm/mcpenetes/internal/log"
)

// RegistryIndex represents the expected structure of the JSON file at a registry URL.
// For now, we assume it's a simple list of version strings.
type RegistryIndex struct {
	Versions []string `json:"versions"` // Example: { "versions": ["1.20.1", "1.19.4"] }
}

// FetchMCPList fetches the list of available MCP versions from a given registry URL.
// It checks the cache first and falls back to HTTP request on miss or expiry.
func FetchMCPList(url string) ([]string, error) {
	// 1. Check cache first
	versions, cacheMiss, err := cache.ReadCache(url) // Use the 3 return values
	if err != nil {
		// Log cache read error but proceed as if it was a miss
		log.Warn("Failed to read cache for %s: %v", url, err) // Use log.Warn
	}
	if !cacheMiss {
		log.Detail("  Cache hit for %s", url) // Use log.Detail for less important info
		return versions, nil
	}

	log.Info("  Cache miss or expired for %s, fetching...", url) // Use log.Info

	// 2. Cache miss or expired, proceed with HTTP fetch
	client := &http.Client{
		Timeout: 10 * time.Second, // Add a timeout to prevent hanging indefinitely
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request for %s: %w", url, err)
	}
	// Set a user-agent? Might be polite.
	req.Header.Set("User-Agent", "mcpetes-cli/0.0.1") // Adjust version as needed

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch from %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch from %s: received status code %d", url, resp.StatusCode)
	}

	var index RegistryIndex
	if err := json.NewDecoder(resp.Body).Decode(&index); err != nil {
		return nil, fmt.Errorf("failed to parse JSON from %s: %w", url, err)
	}

	// 3. Write the fetched result to cache
	if err := cache.WriteCache(url, index.Versions); err != nil {
		// Log cache write error but don't fail the operation
		log.Warn("Failed to write cache for %s: %v", url, err) // Use log.Warn
	}

	return index.Versions, nil
}
