package registry

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/tuannvm/mcpenetes/internal/cache"
	"github.com/tuannvm/mcpenetes/internal/log"
)

// RegistryIndex represents the structure of responses from different registry types
type RegistryIndex struct {
	// Direct version list format
	Versions []string `json:"versions,omitempty"` // Example: { "versions": ["1.20.1", "1.19.4"] }

	// Smithery API format
	SmitheryServers []struct {
		QualifiedName string `json:"qualifiedName"`
		DisplayName   string `json:"displayName"`
		Version       string `json:"version"`
	} `json:"smitheryServers,omitempty"`

	// Glama API format - root level fields
	PageInfo struct {
		EndCursor       string `json:"endCursor"`
		HasNextPage     bool   `json:"hasNextPage"`
		HasPreviousPage bool   `json:"hasPreviousPage"`
		StartCursor     string `json:"startCursor"`
	} `json:"pageInfo,omitempty"`
	Servers []struct {
		ID          string   `json:"id"`
		Name        string   `json:"name"`
		Attributes  []string `json:"attributes"`
		Description string   `json:"description"`
		URL         string   `json:"url"`
		Repository  struct {
			URL string `json:"url"`
		} `json:"repository"`
		SPDXLicense struct {
			Name string `json:"name"`
			URL  string `json:"url"`
		} `json:"spdxLicense"`
		Tools                          []interface{} `json:"tools"`
		EnvironmentVariablesJSONSchema interface{}   `json:"environmentVariablesJsonSchema"`
	} `json:"servers,omitempty"`
}

// formatRegistryURL ensures the registry URL is properly formatted for the specific registry type
func formatRegistryURL(url string) string {
	// Handle Glama API URLs
	if match, _ := regexp.MatchString(`^https?://glama\.ai(/.*)?$`, url); match {
		// If it's a Glama URL, ensure it points to the API endpoint
		baseURL := "https://glama.ai/api/mcp/v1/servers"
		// If additional query parameters were provided, preserve them
		if strings.Contains(url, "?") {
			parts := strings.SplitN(url, "?", 2)
			return baseURL + "?" + parts[1]
		}
		return baseURL
	}
	return url
}

// FetchMCPList fetches the list of available MCP versions from a given registry URL.
// It checks the cache first and falls back to HTTP request on miss or expiry.
func FetchMCPList(url string) ([]string, error) {
	// Format the URL appropriately for the registry type
	url = formatRegistryURL(url)

	// 1. Check cache first
	cachedVersions, cacheMiss, err := cache.ReadCache(url) // Use the 3 return values
	if err != nil {
		// Log cache read error but proceed as if it was a miss
		log.Warn("Failed to read cache for %s: %v", url, err) // Use log.Warn
	}
	if !cacheMiss {
		log.Detail("  Cache hit for %s", url) // Use log.Detail for less important info
		return cachedVersions, nil
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

	// Read the response body into a string for debugging
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body from %s: %w", url, err)
	}
	log.Detail("Response from %s: %s", url, string(body))

	var index RegistryIndex
	if err := json.Unmarshal(body, &index); err != nil {
		return nil, fmt.Errorf("failed to parse JSON from %s: %w", url, err)
	}

	// Debug log the parsed structure
	log.Detail("Parsed response - Versions: %v, Smithery servers: %d, Glama servers: %d",
		len(index.Versions),
		len(index.SmitheryServers),
		len(index.Servers))

	// Extract versions based on the response format
	var versions []string

	if len(index.Versions) > 0 {
		// Direct versions format
		versions = index.Versions
	} else if len(index.SmitheryServers) > 0 {
		// Smithery API format - extract versions from servers
		for _, server := range index.SmitheryServers {
			if server.Version != "" {
				versions = append(versions, server.Version)
			} else {
				versions = append(versions, server.QualifiedName)
			}
		}
	} else if index.Servers != nil {
		// Glama API format - handle pagination
		for _, server := range index.Servers {
			versions = append(versions, fmt.Sprintf("%s: %s", server.Name, server.Description))
		}

		// If there are more pages, fetch them
		cursor := index.PageInfo.EndCursor
		for index.PageInfo.HasNextPage {
			// Construct URL with cursor
			paginatedURL := url
			if cursor != "" {
				if paginatedURL[len(paginatedURL)-1] != '?' {
					paginatedURL += "?"
				}
				paginatedURL += fmt.Sprintf("after=%s&first=100", cursor)
			}

			// Fetch next page
			req, err := http.NewRequest("GET", paginatedURL, nil)
			if err != nil {
				log.Warn("Failed to create request for next page: %v", err)
				break
			}
			req.Header.Set("User-Agent", "mcpetes-cli/0.0.1")

			resp, err := client.Do(req)
			if err != nil {
				log.Warn("Failed to fetch next page: %v", err)
				break
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				log.Warn("Failed to fetch next page: status %d", resp.StatusCode)
				break
			}

			var nextPage RegistryIndex
			if err := json.NewDecoder(resp.Body).Decode(&nextPage); err != nil {
				log.Warn("Failed to parse next page: %v", err)
				break
			}

			// Add servers from this page
			for _, server := range nextPage.Servers {
				serverInfo := server.Name
				if server.Description != "" {
					serverInfo = fmt.Sprintf("%s (%s)", server.Name, server.Description)
				}
				versions = append(versions, serverInfo)
			}

			// Update cursor for next page
			cursor = nextPage.PageInfo.EndCursor
			index.PageInfo.HasNextPage = nextPage.PageInfo.HasNextPage
		}
	}

	if len(versions) == 0 {
		return nil, fmt.Errorf("no versions found in response from %s", url)
	}

	// 3. Write the fetched result to cache
	if err := cache.WriteCache(url, versions); err != nil {
		// Log cache write error but don't fail the operation
		log.Warn("Failed to write cache for %s: %v", url, err) // Use log.Warn
	}

	return versions, nil
}
