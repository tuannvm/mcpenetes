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

	// Determine the target format based on the client or file extension
	format := strings.ToLower(filepath.Ext(clientConfigPath))

	var outputData []byte

	switch format {
	case ".json":
		// Assume client wants the exact MCPServer structure for now
		outputData, err = json.MarshalIndent(serverConf, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal config to JSON for %s: %w", clientName, err)
		}
	case ".yaml", ".yml":
		// Assume client wants the exact MCPServer structure for now
		outputData, err = yaml.Marshal(serverConf)
		if err != nil {
			return fmt.Errorf("failed to marshal config to YAML for %s: %w", clientName, err)
		}
	case ".toml":
		// Assume client wants the exact MCPServer structure for now
		buf := new(bytes.Buffer)
		if err := toml.NewEncoder(buf).Encode(serverConf); err != nil {
			return fmt.Errorf("failed to marshal config to TOML for %s: %w", clientName, err)
		}
		outputData = buf.Bytes()
	default:
		return fmt.Errorf("unsupported config format '%s' for client %s", format, clientName)
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
