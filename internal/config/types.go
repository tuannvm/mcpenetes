package config

// Config represents the structure of config.yaml
type Config struct {
	Version     int               `yaml:"version"`
	Registries  []Registry        `yaml:"registries"`
	SelectedMCP string            `yaml:"selected_mcp"`
	Clients     map[string]Client `yaml:"clients"`
	Backups     BackupConfig      `yaml:"backups"`
}

// Registry defines a registry endpoint
type Registry struct {
	Name string `yaml:"name"`
	URL  string `yaml:"url"`
}

// Client defines a target client configuration location
type Client struct {
	ConfigPath string `yaml:"config_path"`
}

// BackupConfig defines backup settings
type BackupConfig struct {
	Path      string `yaml:"path"`
	Retention int    `yaml:"retention"`
}

// MCPConfig represents the structure of mcp.json
type MCPConfig struct {
	MCPServers map[string]MCPServer `json:"mcpServers"`
}

// MCPServer defines the configuration for a single MCP server
// Note: The exact structure for MCPServer is not fully defined in the README's mcp.json example.
// We'll need to define this based on what information needs to be translated for clients.
// For now, let's add some placeholder fields.
type MCPServer struct {
	Command     string            `json:"command,omitempty"`
	Args        []string          `json:"args,omitempty"`
	Environment map[string]string `json:"environment,omitempty"`
	// Add other necessary fields based on client requirements
}
