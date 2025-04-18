package config

import (
	"encoding/json" // Added json import
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const DefaultConfigFileName = "config.yaml"
const DefaultMCPFileName = "mcp.json"

// getConfigDir returns the application's configuration directory path.
func getConfigDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}
	return filepath.Join(homeDir, ".config", "mcpetes"), nil
}

// Variable to allow mocking in tests
var getConfigPath = func() (string, error) {
	configDir, err := getConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, DefaultConfigFileName), nil
}

// getConfigPaths returns the full paths for the main config file and the mcp config file.
func getConfigPaths() (configFilePath, mcpFilePath string, err error) {
	configDir, err := getConfigDir()
	if err != nil {
		return "", "", err
	}
	configFilePath = filepath.Join(configDir, DefaultConfigFileName)
	mcpFilePath = filepath.Join(configDir, DefaultMCPFileName)
	return configFilePath, mcpFilePath, nil
}

// GetDefaultConfig returns the default configuration structure.
func GetDefaultConfig() *Config {
	// Define default values here
	configDir, _ := getConfigDir()            // Ignore error for default path generation
	backupPath := "~/.config/mcpetes/backups" // Default string
	if configDir != "" {
		backupPath = filepath.Join(configDir, "backups")
	}

	return &Config{
		SelectedMCP: "", // No default selection initially
		Registries: []Registry{
			{
				Name: "glama",
				URL:  "https://glama.ai/api/mcp/v1/servers",
			},
		},
		Clients: make(map[string]Client),
		Backups: BackupConfig{
			Path: backupPath,
		},
	}
}

// LoadConfig loads the application configuration from the default path.
// If the file doesn't exist, it creates a default one.
func LoadConfig() (*Config, error) {
	configFilePath, err := getConfigPath()
	if err != nil {
		return nil, fmt.Errorf("failed to determine config path: %w", err)
	}

	data, err := os.ReadFile(configFilePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// Config file doesn't exist, create a default one
			fmt.Printf("Config file not found at %s. Creating default config.\n", configFilePath)
			defaultCfg := GetDefaultConfig()
			if err := SaveConfig(defaultCfg); err != nil {
				return nil, fmt.Errorf("failed to create default config file: %w", err)
			}
			return defaultCfg, nil
		}
		return nil, fmt.Errorf("failed to read config file '%s': %w", configFilePath, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file '%s': %w", configFilePath, err)
	}

	// Ensure nested maps/slices are initialized if nil (common issue with YAML unmarshaling)
	if cfg.Clients == nil {
		cfg.Clients = make(map[string]Client)
	}
	if len(cfg.Registries) == 0 {
		// Populate default registries if none are configured
		cfg.Registries = GetDefaultConfig().Registries
	}

	return &cfg, nil
}

// SaveConfig saves the application configuration to the default path.
func SaveConfig(cfg *Config) error {
	if cfg == nil {
		return errors.New("cannot save a nil config")
	}
	configFilePath, err := getConfigPath()
	if err != nil {
		return fmt.Errorf("failed to determine config path for saving: %w", err)
	}

	// Ensure the directory exists
	configDir := filepath.Dir(configFilePath)
	if err := os.MkdirAll(configDir, 0750); err != nil { // Use 0750 for config dirs
		return fmt.Errorf("failed to create config directory '%s': %w", configDir, err)
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config to YAML: %w", err)
	}

	if err := os.WriteFile(configFilePath, data, 0600); err != nil { // Use 0600 for config files
		return fmt.Errorf("failed to write config file '%s': %w", configFilePath, err)
	}

	return nil
}

// LoadMCPConfig loads the local MCP configuration file.
func LoadMCPConfig() (*MCPConfig, error) {
	_, mcpFilePath, err := getConfigPaths() // Use the helper
	if err != nil {
		return nil, fmt.Errorf("failed to determine mcp config path: %w", err)
	}

	data, err := os.ReadFile(mcpFilePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// Return an empty config struct if the file doesn't exist
			return &MCPConfig{MCPServers: make(map[string]MCPServer)}, nil
		}
		return nil, fmt.Errorf("failed to read mcp config file '%s': %w", mcpFilePath, err)
	}

	var mcpCfg MCPConfig
	if err := json.Unmarshal(data, &mcpCfg); err != nil { // Use json import
		return nil, fmt.Errorf("failed to parse mcp config file '%s': %w", mcpFilePath, err)
	}

	// Ensure the map is initialized if the file exists but is empty or has null
	if mcpCfg.MCPServers == nil {
		mcpCfg.MCPServers = make(map[string]MCPServer)
	}

	return &mcpCfg, nil
}

// SaveMCPConfig saves the local MCP configuration file.
// Assumes MCPConfig is defined as `type MCPConfig map[string]MCPServer` in types.go
func SaveMCPConfig(mcpCfg *MCPConfig) error {
	if mcpCfg == nil {
		// Or perhaps save an empty map? For now, error out.
		return errors.New("cannot save a nil MCP config")
	}
	_, mcpFilePath, err := getConfigPaths() // Use the helper
	if err != nil {
		return fmt.Errorf("failed to determine mcp config path for saving: %w", err)
	}

	// Ensure the directory exists
	configDir := filepath.Dir(mcpFilePath)
	if err := os.MkdirAll(configDir, 0750); err != nil {
		return fmt.Errorf("failed to create config directory '%s': %w", configDir, err)
	}

	data, err := json.MarshalIndent(mcpCfg, "", "  ") // Use json import and MarshalIndent
	if err != nil {
		return fmt.Errorf("failed to marshal mcp config to JSON: %w", err)
	}

	if err := os.WriteFile(mcpFilePath, data, 0600); err != nil { // Use 0600 for config files
		return fmt.Errorf("failed to write mcp config file '%s': %w", mcpFilePath, err)
	}

	return nil
}
