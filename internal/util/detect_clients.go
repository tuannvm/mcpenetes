package util

import (
	"os"
	"path/filepath"
	"runtime"

	"github.com/tuannvm/mcpenetes/internal/config"
)

// DetectedClient represents a client detected on the user's system
type DetectedClient struct {
	Name       string
	ConfigPath string
}

// DetectMCPClients automatically detects installed MCP-compatible clients
// and their configuration paths on the user's system
func DetectMCPClients() (map[string]config.Client, error) {
	clients := make(map[string]config.Client)
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	// Define potential client paths based on OS
	var clientPaths []struct {
		Name       string
		ConfigDir  string
		ConfigFile string
	}

	switch runtime.GOOS {
	case "darwin": // macOS
		clientPaths = []struct {
			Name       string
			ConfigDir  string
			ConfigFile string
		}{
			// VS Code
			{
				Name:       "vscode",
				ConfigDir:  filepath.Join(homeDir, "Library", "Application Support", "Code", "User"),
				ConfigFile: "settings.json",
			},
			// VS Code Insiders
			{
				Name:       "vscode-insiders",
				ConfigDir:  filepath.Join(homeDir, "Library", "Application Support", "Code - Insiders", "User"),
				ConfigFile: "settings.json",
			},
			// Cursor
			{
				Name:       "cursor",
				ConfigDir:  filepath.Join(homeDir, ".cursor"),
				ConfigFile: "mcp.json",
			},
			// Claude Desktop (exact path from documentation)
			{
				Name:       "claude-desktop",
				ConfigDir:  filepath.Join(homeDir, "Library", "Application Support", "Claude"),
				ConfigFile: "claude_desktop_config.json",
			},
			// Windsurf (exact path from documentation)
			{
				Name:       "windsurf",
				ConfigDir:  filepath.Join(homeDir, ".codeium", "windsurf"),
				ConfigFile: "mcp_config.json",
			},
		}
	case "linux":
		clientPaths = []struct {
			Name       string
			ConfigDir  string
			ConfigFile string
		}{
			// VS Code
			{
				Name:       "vscode",
				ConfigDir:  filepath.Join(homeDir, ".config", "Code", "User"),
				ConfigFile: "settings.json",
			},
			// VS Code Insiders
			{
				Name:       "vscode-insiders",
				ConfigDir:  filepath.Join(homeDir, ".config", "Code - Insiders", "User"),
				ConfigFile: "settings.json",
			},
			// Cursor
			{
				Name:       "cursor",
				ConfigDir:  filepath.Join(homeDir, ".cursor"),
				ConfigFile: "settings.json",
			},
			// Claude Desktop (exact path from documentation)
			{
				Name:       "claude-desktop",
				ConfigDir:  filepath.Join(homeDir, ".config", "Claude"),
				ConfigFile: "claude_desktop_config.json",
			},
			// Windsurf (exact path from documentation)
			{
				Name:       "windsurf",
				ConfigDir:  filepath.Join(homeDir, ".codeium", "windsurf"),
				ConfigFile: "mcp_config.json",
			},
		}
	case "windows":
		appData := os.Getenv("APPDATA")
		userProfile := os.Getenv("USERPROFILE")
		clientPaths = []struct {
			Name       string
			ConfigDir  string
			ConfigFile string
		}{
			// VS Code
			{
				Name:       "vscode",
				ConfigDir:  filepath.Join(appData, "Code", "User"),
				ConfigFile: "settings.json",
			},
			// VS Code Insiders
			{
				Name:       "vscode-insiders",
				ConfigDir:  filepath.Join(appData, "Code - Insiders", "User"),
				ConfigFile: "settings.json",
			},
			// Cursor
			{
				Name:       "cursor",
				ConfigDir:  filepath.Join(appData, "Cursor", "User"),
				ConfigFile: "mcp.json",
			},
			// Claude Desktop (exact path from documentation)
			{
				Name:       "claude-desktop",
				ConfigDir:  filepath.Join(appData, "Claude"),
				ConfigFile: "claude_desktop_config.json",
			},
			// Windsurf (exact path from documentation)
			{
				Name:       "windsurf",
				ConfigDir:  filepath.Join(userProfile, ".codeium", "windsurf"),
				ConfigFile: "mcp_config.json",
			},
		}
	}

	// Check each potential path
	for _, client := range clientPaths {
		configPath := filepath.Join(client.ConfigDir, client.ConfigFile)
		if _, err := os.Stat(configPath); err == nil {
			clients[client.Name] = config.Client{
				ConfigPath: configPath,
			}
		}
	}

	// Also check for directory existence for clients that might not have the file yet
	// This helps with first-time setups
	checkDirs := map[string]string{
		"claude-desktop": filepath.Join(homeDir, "Library", "Application Support", "Claude"),
		"windsurf":       filepath.Join(homeDir, ".codeium", "windsurf"),
	}

	for clientName, dirPath := range checkDirs {
		if _, exists := clients[clientName]; !exists {
			// If we didn't find the client by file check, see if the directory exists
			if _, err := os.Stat(dirPath); err == nil {
				// Directory exists, so add with default config file
				var configFile string
				switch clientName {
				case "claude-desktop":
					configFile = "claude_desktop_config.json"
				case "windsurf":
					configFile = "mcp_config.json"
				default:
					configFile = "settings.json"
				}
				clients[clientName] = config.Client{
					ConfigPath: filepath.Join(dirPath, configFile),
				}
			}
		}
	}

	return clients, nil
}
