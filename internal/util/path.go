package util

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ExpandPath expands tilde (~) in a path to the user's home directory.
func ExpandPath(path string) (string, error) {
	if !strings.HasPrefix(path, "~") {
		return path, nil // No tilde prefix, return as is
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}

	return filepath.Join(homeDir, path[1:]), nil
}
