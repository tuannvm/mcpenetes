package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors" // Import standard errors package
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

// Helper function to calculate expected hash for testing getCachePath
func expectedCacheFilename(registryURL string) string {
	hasher := sha256.New()
	hasher.Write([]byte(registryURL))
	hash := hex.EncodeToString(hasher.Sum(nil))
	return fmt.Sprintf("%s.json", hash)
}

// Test that getCachePath correctly generates cache file paths from registry URLs
func TestGetCachePath(t *testing.T) {
	// Test cases
	testCases := []struct {
		name           string
		registryURL    string
		wantErr        bool
		expectedBaseFn string // The expected base filename (e.g., <hash>.json)
	}{
		{
			name:           "Simple URL",
			registryURL:    "https://example.com/index.json",
			wantErr:        false,
			expectedBaseFn: expectedCacheFilename("https://example.com/index.json"), // Use helper
		},
		{
			name:           "URL with query parameters",
			registryURL:    "https://example.com/api?version=latest",
			wantErr:        false,
			expectedBaseFn: expectedCacheFilename("https://example.com/api?version=latest"), // Use helper
		},
		{
			name:           "Invalid URL format (no scheme/host)", // Adjusted description
			registryURL:    "nodomain",                            // Changed to a more realistic invalid format case
			wantErr:        true,
			expectedBaseFn: "",
		},
		{
			name:           "Invalid URL parsing", // Added case for unparsable URL
			registryURL:    "://invalid-url",
			wantErr:        true,
			expectedBaseFn: "",
		},
	}

	// Temporarily override cacheDirPath for predictable test paths
	originalCacheDirPath := cacheDirPath
	testBaseDir := t.TempDir() // Use a test-specific temp dir for the base
	cacheDirPath = testBaseDir // Set the global variable used by the real getCachePath
	defer func() {
		cacheDirPath = originalCacheDirPath // Restore original cacheDirPath
	}()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// We test the *real* getCachePath function here, just controlling its base directory
			path, err := getCachePath(tc.registryURL) // Use the real function

			// Check error expectations
			if tc.wantErr {
				if err == nil {
					t.Errorf("Expected error for URL '%s', got none", tc.registryURL)
				}
				return // Don't check path if error was expected
			}

			if err != nil {
				t.Fatalf("Unexpected error for URL '%s': %v", tc.registryURL, err)
			}

			// Check the path structure
			if !filepath.IsAbs(path) {
				t.Errorf("Expected absolute path, got: %s", path)
			}
			// Check it's within our controlled test cache dir
			if !strings.HasPrefix(path, testBaseDir) {
				t.Errorf("Path '%s' is not within the expected test cache directory '%s'", path, testBaseDir)
			}
			// Check the base filename uses the hash
			baseFn := filepath.Base(path)
			if baseFn != tc.expectedBaseFn {
				t.Errorf("Expected base filename '%s', got '%s'", tc.expectedBaseFn, baseFn)
			}
		})
	}
}

// Helper function to create a temp cache file
func createTempCacheFile(t *testing.T, timestamp time.Time, versions []string) (string, string) {
	t.Helper()

	// Create a temporary directory for our test cache
	tempDir := t.TempDir()

	// Create test cache entry
	entry := CacheEntry{
		Timestamp: timestamp,
		Versions:  versions,
	}

	data, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal test cache entry: %v", err)
	}

	// Create directory structure
	cachePath := filepath.Join(tempDir, "test-cache.json")
	if err := os.WriteFile(cachePath, data, 0600); err != nil {
		t.Fatalf("Failed to write test cache file: %v", err)
	}

	return tempDir, cachePath
}

// Test the ReadCache function with various scenarios
func TestReadCache(t *testing.T) {
	// Save the original function and restore it after the test
	originalGetCachePath := getCachePath
	defer func() { getCachePath = originalGetCachePath }()

	// Test cases
	testCases := []struct {
		name          string
		setupFunc     func() string // Returns the path to be used by the mock getCachePath
		registryURL   string        // Used only for descriptive purposes in this test
		wantVersions  []string
		wantCacheMiss bool
		wantErr       bool
	}{
		{
			name: "Valid unexpired cache",
			setupFunc: func() string {
				_, path := createTempCacheFile(t, time.Now(), []string{"1.0.0", "2.0.0"})
				return path // Ensure path is returned
			},
			registryURL:   "https://example.com/index.json",
			wantVersions:  []string{"1.0.0", "2.0.0"},
			wantCacheMiss: false,
			wantErr:       false,
		},
		{
			name: "Expired cache",
			setupFunc: func() string {
				// Expire slightly more than cacheTTL to avoid flakes
				_, path := createTempCacheFile(t, time.Now().Add(-(cacheTTL + 5*time.Second)), []string{"old"})
				return path // Ensure path is returned
			},
			registryURL:   "https://example.com/expired.json",
			wantVersions:  nil,
			wantCacheMiss: true, // Cache miss due to expiry
			wantErr:       false,
		},
		{
			name: "Non-existent cache file",
			setupFunc: func() string {
				// Return a path that doesn't exist within a temp dir
				return filepath.Join(t.TempDir(), "nonexistent-cache.json")
			},
			registryURL:   "https://example.com/nonexistent.json",
			wantVersions:  nil,
			wantCacheMiss: true,  // Cache miss because file doesn't exist
			wantErr:       false, // os.IsNotExist errors are handled as cache misses, not errors
		},
		{
			name: "Invalid JSON in cache file",
			setupFunc: func() string {
				tempDir := t.TempDir()
				invalidPath := filepath.Join(tempDir, "invalid-cache.json")
				// Write invalid JSON content
				err := os.WriteFile(invalidPath, []byte(`{"timestamp":"not-a-time", "versions":["bad"]`), 0600)
				if err != nil {
					t.Fatalf("Failed to write invalid cache file: %v", err)
				}
				return invalidPath // Ensure path is returned
			},
			registryURL:   "https://example.com/invalid.json",
			wantVersions:  nil,
			wantCacheMiss: true,  // Cache miss due to unmarshal error
			wantErr:       false, // Unmarshal errors are handled as cache misses, not errors
		},
		{
			name: "Error from getCachePath", // Add test case for getCachePath error
			setupFunc: func() string {
				// This path isn't actually used because getCachePath will error
				return ""
			},
			registryURL:   "://force-error", // Invalid URL to make getCachePath error
			wantVersions:  nil,
			wantCacheMiss: true, // Treat getCachePath error as a cache miss scenario for simplicity
			wantErr:       true, // Expect the underlying error from getCachePath
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mock for getCachePath for this specific test case
			if tc.name == "Error from getCachePath" {
				// Restore the original getCachePath temporarily to test its error path
				getCachePath = originalGetCachePath
			} else {
				// For other cases, mock it to return the path from setupFunc
				cachePath := tc.setupFunc()
				getCachePath = func(string) (string, error) {
					return cachePath, nil
				}
			}

			// Call the function under test
			versions, cacheMiss, err := ReadCache(tc.registryURL)

			// Restore the mock immediately after the call if we changed it
			if tc.name == "Error from getCachePath" {
				getCachePath = originalGetCachePath // Put the original back if we used it
			}

			// Check results
			if tc.wantErr {
				if err == nil {
					t.Errorf("Expected error, got none")
				}
				// Optionally check the error type or message if needed
			} else if err != nil {
				// Don't fail on os.IsNotExist or JSON errors, as they are treated as cache misses
				if !os.IsNotExist(err) && !errors.Is(err, &json.SyntaxError{}) && !errors.Is(err, &json.UnmarshalTypeError{}) {
					t.Fatalf("Unexpected error: %v", err)
				}
				// If it was a handled error (like file not found or bad JSON), ensure cacheMiss is true
				if !cacheMiss {
					t.Errorf("Expected cacheMiss to be true on handled error, but got false. Error: %v", err)
				}
			}

			// Check cacheMiss status consistency
			if cacheMiss != tc.wantCacheMiss {
				t.Errorf("Expected cacheMiss %v, got %v", tc.wantCacheMiss, cacheMiss)
			}

			// Only compare versions if it wasn't a cache miss and no error occurred (or error was expected)
			if !tc.wantCacheMiss && !tc.wantErr {
				if !reflect.DeepEqual(versions, tc.wantVersions) {
					t.Errorf("Expected versions %v, got %v", tc.wantVersions, versions)
				}
			} else if versions != nil { // If it was a cache miss or error, versions should be nil
				t.Errorf("Expected nil versions on cache miss or error, got %v", versions)
			}

			// Crucially, restore the default mock behavior for the next iteration
			// (or restore the original if that was the default for the test)
			// This defer in the outer function handles the final cleanup,
			// but we need to reset between test cases if we modified getCachePath inside the loop.
			// Re-assigning the mock function *before* the call handles this correctly now.
		})
	}
	// Final cleanup is handled by the defer at the start of TestReadCache
}

// Test the WriteCache function
func TestWriteCache(t *testing.T) {
	// Save the original function and restore it after
	originalGetCachePath := getCachePath
	defer func() { getCachePath = originalGetCachePath }()

	testCases := []struct {
		name        string
		registryURL string
		versions    []string
		mockSetup   func() (mockFunc func(string) (string, error), cleanup func()) // Returns mock and cleanup
		wantErr     bool
	}{
		{
			name:        "Successful write",
			registryURL: "https://example.com/write-ok.json",
			versions:    []string{"1.0.0", "2.0.0"},
			mockSetup: func() (func(string) (string, error), func()) {
				tempDir := t.TempDir()
				cacheFilePath := filepath.Join(tempDir, expectedCacheFilename("https://example.com/write-ok.json"))
				mock := func(string) (string, error) {
					return cacheFilePath, nil
				}
				// No specific cleanup needed beyond t.TempDir()
				return mock, func() {}
			},
			wantErr: false,
		},
		{
			name:        "Error from getCachePath",
			registryURL: "://bad-url-for-write",
			versions:    []string{"1.0.0"},
			mockSetup: func() (func(string) (string, error), func()) {
				// Use the original getCachePath to trigger its internal error
				mock := func(regURL string) (string, error) {
					// Temporarily restore original cacheDirPath if needed, or just call original
					// Assuming original getCachePath handles its own dir path correctly
					return originalGetCachePath(regURL)
				}
				// No specific cleanup needed
				return mock, func() {}
			},
			wantErr: true, // Expect error from getCachePath
		},
		{
			name:        "Error creating directory (permission denied simulation)",
			registryURL: "https://example.com/write-perm-error.json",
			versions:    []string{"1.0.0"},
			mockSetup: func() (func(string) (string, error), func()) {
				// Create a read-only directory
				readOnlyDir := filepath.Join(t.TempDir(), "read-only-dir")
				if err := os.Mkdir(readOnlyDir, 0555); err != nil { // Read/execute only
					t.Fatalf("Failed to create read-only dir: %v", err)
				}
				cacheFilePath := filepath.Join(readOnlyDir, "subdir", expectedCacheFilename("https://example.com/write-perm-error.json"))

				mock := func(string) (string, error) {
					// This path requires creating "subdir" inside "read-only-dir", which should fail
					return cacheFilePath, nil
				}
				cleanup := func() {
					// Attempt to make writable to allow removal by t.TempDir cleanup
					_ = os.Chmod(readOnlyDir, 0755)
				}
				return mock, cleanup
			},
			wantErr: true, // Expect error during os.MkdirAll in WriteCache
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockFunc, cleanup := tc.mockSetup()
			getCachePath = mockFunc // Set the mock for this test case
			defer cleanup()         // Run cleanup after the test case

			// Call WriteCache
			err := WriteCache(tc.registryURL, tc.versions)

			// Check for errors
			if tc.wantErr {
				if err == nil {
					t.Errorf("Expected error, got none")
				}
				// Restore mock before next iteration (handled by outer defer)
				getCachePath = originalGetCachePath // Restore here for safety between iterations
				return
			} else if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			// If no error was expected, verify the file was written correctly
			// Need the path again to read the file
			cacheFilePath, pathErr := getCachePath(tc.registryURL)
			if pathErr != nil {
				t.Fatalf("Failed to get cache path for verification: %v", pathErr)
			}

			data, readErr := os.ReadFile(cacheFilePath)
			if readErr != nil {
				t.Fatalf("Failed to read cache file '%s' after writing: %v", cacheFilePath, readErr)
			}

			var entry CacheEntry
			if unmarshalErr := json.Unmarshal(data, &entry); unmarshalErr != nil {
				t.Fatalf("Failed to unmarshal cache entry from '%s': %v", cacheFilePath, unmarshalErr)
			}

			if !reflect.DeepEqual(entry.Versions, tc.versions) {
				t.Errorf("Expected versions %v, got %v", tc.versions, entry.Versions)
			}

			// Verify the timestamp is recent (within a reasonable threshold like 5 seconds)
			if time.Since(entry.Timestamp) > 5*time.Second {
				t.Errorf("Timestamp is too old: %v (written), now %v", entry.Timestamp, time.Now())
			}

			// Restore mock before next iteration (handled by outer defer)
			getCachePath = originalGetCachePath // Restore here for safety between iterations
		})
	}
	// Final cleanup is handled by the defer at the start of TestWriteCache
}
