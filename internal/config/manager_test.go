package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"gopkg.in/yaml.v3"
)

// Helper function to create a temporary config file
func createTempConfigFile(t *testing.T, dir string, filename string, content string) string {
	t.Helper()
	tmpFilePath := filepath.Join(dir, filename)
	err := os.WriteFile(tmpFilePath, []byte(content), 0600)
	if err != nil {
		t.Fatalf("Failed to write temporary config file '%s': %v", tmpFilePath, err)
	}
	return tmpFilePath
}

func TestLoadConfig(t *testing.T) {
	t.Run("Load existing config", func(t *testing.T) {
		tempDir := t.TempDir()
		expectedConfig := &Config{
			Version: 1,
			Registries: []Registry{
				{Name: "official", URL: "http://example.com/index.json"},
			},
			MCPs: []string{"server-a"},
			Clients: map[string]Client{
				"cursor": {ConfigPath: "~/.cursor/config.json"},
			},
			Backups: BackupConfig{
				Path: "~/.config/mcpetes/backups",
			},
		}
		yamlData, _ := yaml.Marshal(expectedConfig)
		// Pass DefaultConfigFileName explicitly
		tempConfigFile := createTempConfigFile(t, tempDir, DefaultConfigFileName, string(yamlData))

		// Temporarily override the config path function
		originalGetConfigPath := getConfigPath // Now refers to the var
		getConfigPath = func() (string, error) {
			return tempConfigFile, nil
		}
		defer func() { getConfigPath = originalGetConfigPath }() // Restore original

		loadedConfig, err := LoadConfig()
		if err != nil {
			t.Fatalf("LoadConfig failed: %v", err)
		}

		if !reflect.DeepEqual(loadedConfig, expectedConfig) {
			t.Errorf("Loaded config does not match expected.\nExpected: %+v\nGot:      %+v", expectedConfig, loadedConfig)
		}
	})

	t.Run("Load non-existent config creates default", func(t *testing.T) {
		tempDir := t.TempDir()
		// Use DefaultConfigFileName constant
		nonExistentPath := filepath.Join(tempDir, DefaultConfigFileName)

		// Temporarily override the config path function
		originalGetConfigPath := getConfigPath // Now refers to the var
		getConfigPath = func() (string, error) {
			return nonExistentPath, nil
		}
		defer func() { getConfigPath = originalGetConfigPath }() // Restore original

		loadedConfig, err := LoadConfig()
		if err != nil {
			t.Fatalf("LoadConfig failed for non-existent file: %v", err)
		}

		// Check if the file was created
		if _, err := os.Stat(nonExistentPath); os.IsNotExist(err) {
			t.Errorf("Config file was not created at %s", nonExistentPath)
		}

		// Check if loaded config is the default one
		// Use GetDefaultConfig function
		defaultConfig := GetDefaultConfig()
		if !reflect.DeepEqual(loadedConfig, defaultConfig) {
			t.Errorf("Loaded config is not the default one.\nExpected: %+v\nGot:      %+v", defaultConfig, loadedConfig)
		}
	})

	// Add more tests: invalid YAML format, permission errors (harder to test reliably)
}

func TestSaveConfig(t *testing.T) {
	tempDir := t.TempDir()
	savePath := filepath.Join(tempDir, "test_config_save.yaml")

	configToSave := &Config{
		Version: 1,
		MCPs: []string{"server-b"},
		Registries: []Registry{{Name: "local", URL: "file:///tmp/index.json"}},
		Clients:    map[string]Client{"vscode": {ConfigPath: "~/.config/Code/User/settings.json"}},
		Backups:    BackupConfig{Path: "/tmp/mcpetes_backups"},
	}

	// Temporarily override the config path function
	originalGetConfigPath := getConfigPath // Now refers to the var
	getConfigPath = func() (string, error) {
		return savePath, nil
	}
	defer func() { getConfigPath = originalGetConfigPath }() // Restore original

	err := SaveConfig(configToSave)
	if err != nil {
		t.Fatalf("SaveConfig failed: %v", err)
	}

	// Read the saved file back
	savedData, err := os.ReadFile(savePath)
	if err != nil {
		t.Fatalf("Failed to read back saved config file: %v", err)
	}

	var loadedConfig Config
	err = yaml.Unmarshal(savedData, &loadedConfig)
	if err != nil {
		t.Fatalf("Failed to unmarshal saved config data: %v", err)
	}

	if !reflect.DeepEqual(&loadedConfig, configToSave) {
		t.Errorf("Saved config does not match original.\nExpected: %+v\nGot:      %+v", configToSave, &loadedConfig)
	}

	// Test saving nil config (should probably error or save default? Let's assume error)
	err = SaveConfig(nil)
	if err == nil {
		t.Errorf("Expected error when saving nil config, but got nil")
	}
}
