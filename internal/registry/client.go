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

// ServerData represents information about an MCP server
type ServerData struct {
	Name          string
	Description   string
	RepositoryURL string
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
				versions = append(versions, fmt.Sprintf("%s: %s", server.Name, server.Description))
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

// FetchMCPServersWithCache fetches server information from a registry URL, using cache when available.
// Accepts a forceRefresh parameter to bypass the cache when needed.
func FetchMCPServersWithCache(registryURL string, forceRefresh bool) ([]ServerData, error) {
	// Format the URL appropriately for the registry type
	url := formatRegistryURL(registryURL)
	
	// Check cache first (unless forceRefresh is true)
	if !forceRefresh {
		cachedServers, cacheMiss, err := cache.ReadServerCache(url)
		if err != nil {
			// Log cache read error but proceed as if it was a miss
			log.Warn("Failed to read server cache for %s: %v", url, err)
		}
		if !cacheMiss {
			log.Detail("  Cache hit for server data from %s", url)
			
			// Convert cached data to ServerData format
			servers := make([]ServerData, len(cachedServers))
			for i, s := range cachedServers {
				servers[i] = ServerData{
					Name:          s.Name,
					Description:   s.Description,
					RepositoryURL: s.RepositoryURL,
				}
			}
			return servers, nil
		}
		log.Info("  Server cache miss or expired for %s, fetching...", url)
	} else {
		log.Info("  Forcing refresh of server data from %s", url)
	}
	
	// Cache miss, expiry, or forced refresh - fetch from network
	servers, err := FetchMCPServers(url)
	if err != nil {
		return nil, err
	}
	
	// Convert to cache format and save to cache
	cacheServers := make([]cache.ServerInfo, len(servers))
	for i, s := range servers {
		cacheServers[i] = cache.ServerInfo{
			Name:          s.Name,
			Description:   s.Description,
			RepositoryURL: s.RepositoryURL,
		}
	}
	
	// Write to cache
	if err := cache.WriteServerCache(url, cacheServers); err != nil {
		log.Warn("Failed to write server cache for %s: %v", url, err)
	}
	
	return servers, nil
}

// FetchMCPServers fetches server information from a registry URL.
// Similar to FetchMCPList but returns ServerData objects with repository URLs.
func FetchMCPServers(url string) ([]ServerData, error) {
	// Format the URL appropriately for the registry type
	url = formatRegistryURL(url)

	// We'll skip the cache for this function since we need detailed server data

	// Proceed with HTTP fetch
	client := &http.Client{
		Timeout: 10 * time.Second, // Add a timeout to prevent hanging indefinitely
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request for %s: %w", url, err)
	}
	req.Header.Set("User-Agent", "mcpetes-cli/0.0.1")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch from %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch from %s: received status code %d", url, resp.StatusCode)
	}

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body from %s: %w", url, err)
	}

	var index RegistryIndex
	if err := json.Unmarshal(body, &index); err != nil {
		return nil, fmt.Errorf("failed to parse JSON from %s: %w", url, err)
	}

	// Extract server data based on the response format
	var servers []ServerData

	if len(index.Versions) > 0 {
		// Direct versions format - no repository URLs available
		for _, version := range index.Versions {
			servers = append(servers, ServerData{
				Name:          version,
				Description:   "",
				RepositoryURL: "",
			})
		}
	} else if len(index.SmitheryServers) > 0 {
		// Smithery API format - extract server info
		for _, server := range index.SmitheryServers {
			name := server.DisplayName
			if name == "" {
				name = server.QualifiedName
			}

			servers = append(servers, ServerData{
				Name:          name,
				Description:   server.Version,
				RepositoryURL: "", // No repository URL in Smithery format
			})
		}
	} else if index.Servers != nil {
		// Glama API format - handle pagination
		for _, server := range index.Servers {
			repoURL := ""
			if server.Repository.URL != "" {
				repoURL = server.Repository.URL
			}

			servers = append(servers, ServerData{
				Name:          server.Name,
				Description:   server.Description,
				RepositoryURL: repoURL,
			})
		}

		// If there are more pages, fetch them
		cursor := index.PageInfo.EndCursor
		for index.PageInfo.HasNextPage {
			// Construct URL with cursor
			paginatedURL := url
			if cursor != "" {
				if !strings.Contains(paginatedURL, "?") {
					paginatedURL += "?"
				} else if !strings.HasSuffix(paginatedURL, "?") && !strings.HasSuffix(paginatedURL, "&") {
					paginatedURL += "&"
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
				repoURL := ""
				if server.Repository.URL != "" {
					repoURL = server.Repository.URL
				}

				servers = append(servers, ServerData{
					Name:          server.Name,
					Description:   server.Description,
					RepositoryURL: repoURL,
				})
			}

			// Update cursor for next page
			cursor = nextPage.PageInfo.EndCursor
			index.PageInfo.HasNextPage = nextPage.PageInfo.HasNextPage
		}
	}

	if len(servers) == 0 {
		return nil, fmt.Errorf("no servers found in response from %s", url)
	}

	return servers, nil
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
