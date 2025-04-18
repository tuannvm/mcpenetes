package translator

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/tuannvm/mcpenetes/internal/config"
	"github.com/tuannvm/mcpenetes/internal/util"
	"gopkg.in/yaml.v3"
)

// Translator handles backing up and translating MCP configs for clients.
type Translator struct {
	AppConfig *config.Config
	MCPConfig *config.MCPConfig
}

// NewTranslator creates a new Translator instance.
func NewTranslator(appCfg *config.Config, mcpCfg *config.MCPConfig) *Translator {
	return &Translator{
		AppConfig: appCfg,
		MCPConfig: mcpCfg,
	}
}

// BackupClientConfig creates a timestamped backup of a client's configuration file.
func (t *Translator) BackupClientConfig(clientName string, clientConf config.Client) (string, error) {
	backupDir, err := util.ExpandPath(t.AppConfig.Backups.Path)
	if err != nil {
		return "", fmt.Errorf("failed to expand backup path '%s': %w", t.AppConfig.Backups.Path, err)
	}

	clientConfigPath, err := util.ExpandPath(clientConf.ConfigPath)
	if err != nil {
		return "", fmt.Errorf("failed to expand client config path '%s' for %s: %w", clientConf.ConfigPath, clientName, err)
	}

	// Ensure the main backup directory exists
	if err := os.MkdirAll(backupDir, 0750); err != nil {
		return "", fmt.Errorf("failed to create backup directory '%s': %w", backupDir, err)
	}

	// Check if source file exists
	srcInfo, err := os.Stat(clientConfigPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Source config doesn't exist, nothing to back up
			return "", nil // Not an error, just nothing to do
		}
		return "", fmt.Errorf("failed to stat source config file '%s': %w", clientConfigPath, err)
	}
	if srcInfo.IsDir() {
		return "", fmt.Errorf("source config path '%s' is a directory, not a file", clientConfigPath)
	}

	// Create timestamped backup filename
	timestamp := time.Now().Format("20060102-150405") // YYYYMMDD-HHMMSS
	backupFileName := fmt.Sprintf("%s-%s%s", clientName, timestamp, filepath.Ext(clientConfigPath))
	backupFilePath := filepath.Join(backupDir, backupFileName)

	// Open source file
	srcFile, err := os.Open(clientConfigPath)
	if err != nil {
		return "", fmt.Errorf("failed to open source config file '%s': %w", clientConfigPath, err)
	}
	defer srcFile.Close()

	// Create destination backup file
	dstFile, err := os.Create(backupFilePath)
	if err != nil {
		return "", fmt.Errorf("failed to create backup file '%s': %w", backupFilePath, err)
	}
	defer dstFile.Close()

	// Copy content
	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return "", fmt.Errorf("failed to copy config to backup file '%s': %w", backupFilePath, err)
	}

	fmt.Printf("  Backed up '%s' to '%s'\n", clientConfigPath, backupFilePath)

	// TODO: Implement backup retention logic here or separately

	return backupFilePath, nil
}

// TranslateAndApply translates the selected MCP config and writes it to the client's path.
func (t *Translator) TranslateAndApply(clientName string, clientConf config.Client, serverConf config.MCPServer) error {
	clientConfigPath, err := util.ExpandPath(clientConf.ConfigPath)
	if err != nil {
		return fmt.Errorf("failed to expand client config path '%s' for %s: %w", clientConf.ConfigPath, clientName, err)
	}

	fmt.Printf("  Translating config for %s ('%s')...\n", clientName, clientConfigPath)

	// Find the server ID (key) from the MCPConfig map by matching the server config
	serverID := ""
	for id, server := range t.MCPConfig.MCPServers {
		// Compare the relevant fields to find a match
		if server.Command == serverConf.Command &&
			server.URL == serverConf.URL &&
			fmt.Sprintf("%v", server.Args) == fmt.Sprintf("%v", serverConf.Args) &&
			fmt.Sprintf("%v", server.Env) == fmt.Sprintf("%v", serverConf.Env) {
			serverID = id
			break
		}
	}

	if serverID == "" {
		// Fallback: generate a server ID based on command or URL
		if serverConf.Command != "" {
			serverID = strings.Split(serverConf.Command, " ")[0]
		} else if serverConf.URL != "" {
			// Extract domain from URL
			parts := strings.Split(strings.TrimPrefix(strings.TrimPrefix(serverConf.URL, "https://"), "http://"), "/")
			if len(parts) > 0 {
				serverID = parts[0]
			} else {
				serverID = "mcp-server"
			}
		} else {
			serverID = "mcp-server"
		}
	}

	// Determine how to format the config based on client name and file extension
	var outputData []byte
	format := strings.ToLower(filepath.Ext(clientConfigPath))

	// Check if the file already exists to determine if we need to merge with existing config
	existingConfig := make(map[string]interface{})
	existingFile, err := os.ReadFile(clientConfigPath)
	var configExists bool = false
	if err == nil && len(existingFile) > 0 {
		configExists = true
		err = json.Unmarshal(existingFile, &existingConfig)
		if err != nil {
			// File exists but isn't valid JSON, we'll just overwrite it
			configExists = false
		}
	}

	// Prepare the server configuration based on client type
	switch {
	case strings.Contains(clientName, "claude-desktop"):
		// Format expected by Claude Desktop: {"mcpServers": {"server-id": {...server config...}}}
		var claudeConfig map[string]interface{}

		if configExists {
			claudeConfig = existingConfig
		} else {
			claudeConfig = make(map[string]interface{})
			claudeConfig["mcpServers"] = make(map[string]interface{})
		}

		// Check if mcpServers map exists
		mcpServers, ok := claudeConfig["mcpServers"].(map[string]interface{})
		if !ok {
			// Initialize or reset the mcpServers map if it doesn't exist or has wrong type
			mcpServers = make(map[string]interface{})
		}

		// Create a server entry for Claude Desktop format
		serverEntry := make(map[string]interface{})

		// Copy the basic server properties
		if serverConf.Command != "" {
			serverEntry["command"] = serverConf.Command
		}

		if len(serverConf.Args) > 0 {
			serverEntry["args"] = serverConf.Args
		}

		if len(serverConf.Env) > 0 {
			serverEntry["env"] = serverConf.Env
		}

		if serverConf.URL != "" {
			serverEntry["url"] = serverConf.URL
		}

		// Include disabled and autoApprove fields if they're set
		if serverConf.Disabled {
			serverEntry["disabled"] = serverConf.Disabled
		}

		if len(serverConf.AutoApprove) > 0 {
			serverEntry["autoApprove"] = serverConf.AutoApprove
		} else {
			serverEntry["autoApprove"] = []string{}
		}

		// Add/update the server in the map
		mcpServers[serverID] = serverEntry
		claudeConfig["mcpServers"] = mcpServers

		// Marshal the updated config
		outputData, err = json.MarshalIndent(claudeConfig, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal Claude Desktop config: %w", err)
		}

	case strings.Contains(clientName, "windsurf"):
		// Format expected by Windsurf: structure not fully documented
		// Based on documentation sample, create a similar structure for Windsurf
		var windsurfConfig map[string]interface{}

		if configExists {
			windsurfConfig = existingConfig
		} else {
			windsurfConfig = make(map[string]interface{})
		}

		// For Windsurf, we'll use the existing format but ensure we add/update servers
		if _, ok := windsurfConfig["servers"]; !ok {
			windsurfConfig["servers"] = make(map[string]interface{})
		}

		servers, ok := windsurfConfig["servers"].(map[string]interface{})
		if !ok {
			// If the servers key exists but is not a map, create a new one
			servers = make(map[string]interface{})
		}

		// Create or update server entry
		serverEntry := make(map[string]interface{})

		if serverConf.Command != "" {
			serverEntry["command"] = serverConf.Command
		}

		if len(serverConf.Args) > 0 {
			serverEntry["args"] = serverConf.Args
		}

		if len(serverConf.Env) > 0 {
			serverEntry["env"] = serverConf.Env
		}

		if serverConf.URL != "" {
			serverEntry["url"] = serverConf.URL
		}

		// Add/update the server in the map
		servers[serverID] = serverEntry
		windsurfConfig["servers"] = servers

		// Marshal the updated config
		outputData, err = json.MarshalIndent(windsurfConfig, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal Windsurf config: %w", err)
		}

	case strings.Contains(clientName, "vscode") || strings.Contains(clientName, "cursor"):
		// VS Code and Cursor use the same format for MCP servers
		var vscodeConfig map[string]interface{}

		if configExists {
			vscodeConfig = existingConfig
		} else {
			vscodeConfig = make(map[string]interface{})
		}

		// Get or create the mcp.servers object
		mcpServers, ok := vscodeConfig["mcp.servers"].(map[string]interface{})
		if !ok {
			// If mcp.servers doesn't exist or isn't a map, create it
			mcpServers = make(map[string]interface{})
		}

		// Create or update the server config
		serverEntry := make(map[string]interface{})

		if serverConf.Command != "" {
			serverEntry["command"] = serverConf.Command
		}

		if len(serverConf.Args) > 0 {
			serverEntry["args"] = serverConf.Args
		}

		// VSCode format doesn't seem to include env vars in standard format
		if len(serverConf.Env) > 0 {
			serverEntry["env"] = serverConf.Env
		}

		if serverConf.URL != "" {
			serverEntry["url"] = serverConf.URL
		}

		// Add/update the server in the map
		mcpServers[serverID] = serverEntry
		vscodeConfig["mcp.servers"] = mcpServers

		// Marshal the updated config
		outputData, err = json.MarshalIndent(vscodeConfig, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal VS Code/Cursor config: %w", err)
		}

	default:
		// For unknown clients, use a generic approach based on file extension
		switch format {
		case ".json":
			// If we don't know the client, use a standard format that matches our internal structure
			// Our standard format uses mcpServers map like our current config
			var genericConfig map[string]interface{}

			if configExists {
				genericConfig = existingConfig
			} else {
				genericConfig = make(map[string]interface{})
			}

			// Get or create the mcpServers map
			mcpServers, ok := genericConfig["mcpServers"].(map[string]interface{})
			if !ok {
				mcpServers = make(map[string]interface{})
			}

			// Convert serverConf to a map to add to mcpServers
			serverMap := make(map[string]interface{})

			if serverConf.Command != "" {
				serverMap["command"] = serverConf.Command
			}

			if len(serverConf.Args) > 0 {
				serverMap["args"] = serverConf.Args
			}

			if len(serverConf.Env) > 0 {
				serverMap["env"] = serverConf.Env
			}

			if serverConf.URL != "" {
				serverMap["url"] = serverConf.URL
			}

			// Add disabled and autoApprove fields if they're set
			if serverConf.Disabled {
				serverMap["disabled"] = serverConf.Disabled
			}

			if len(serverConf.AutoApprove) > 0 {
				serverMap["autoApprove"] = serverConf.AutoApprove
			} else {
				serverMap["autoApprove"] = []string{}
			}

			// Add the server to the map
			mcpServers[serverID] = serverMap
			genericConfig["mcpServers"] = mcpServers

			outputData, err = json.MarshalIndent(genericConfig, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal generic JSON config: %w", err)
			}

		case ".yaml", ".yml":
			// Create a map for the server with its ID as key
			serverMap := make(map[string]config.MCPServer)
			serverMap[serverID] = serverConf

			outputData, err = yaml.Marshal(serverMap)
			if err != nil {
				return fmt.Errorf("failed to marshal config to YAML for %s: %w", clientName, err)
			}

		case ".toml":
			// Create a map for the server with its ID as key
			serverMap := make(map[string]config.MCPServer)
			serverMap[serverID] = serverConf

			buf := new(bytes.Buffer)
			if err := toml.NewEncoder(buf).Encode(serverMap); err != nil {
				return fmt.Errorf("failed to marshal config to TOML for %s: %w", clientName, err)
			}
			outputData = buf.Bytes()

		default:
			return fmt.Errorf("unsupported config format '%s' for client %s", format, clientName)
		}
	}

	// Ensure the target directory exists
	clientConfigDir := filepath.Dir(clientConfigPath)
	if err := os.MkdirAll(clientConfigDir, 0750); err != nil {
		return fmt.Errorf("failed to create directory '%s' for client %s: %w", clientConfigDir, clientName, err)
	}

	// Write the translated config file
	if err := os.WriteFile(clientConfigPath, outputData, 0644); err != nil { // Use 0644 for client configs generally
		return fmt.Errorf("failed to write config file '%s' for client %s: %w", clientConfigPath, clientName, err)
	}

	fmt.Printf("  Successfully wrote config for %s to '%s'\n", clientName, clientConfigPath)
	return nil
}

// TODO: Implement backup retention cleanup
// func (t *Translator) CleanupBackups() error { ... }
