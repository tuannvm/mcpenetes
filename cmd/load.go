package cmd

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tuannvm/mcpenetes/internal/config"
	"github.com/tuannvm/mcpenetes/internal/log"
)

// loadCmd represents the load command
var loadCmd = &cobra.Command{
	Use:   "load",
	Short: "Load MCP server configuration from clipboard",
	Long:  `Loads MCP server configuration from the clipboard and adds it to mcp.json`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Info("Reading configuration from clipboard...")

		// Get clipboard content
		clipboardContent, err := getClipboard()
		if err != nil {
			log.Fatal("Failed to read clipboard: %v", err)
			return
		}

		if clipboardContent == "" {
			log.Fatal("Clipboard is empty")
			return
		}

		// Parse clipboard content as JSON
		var clipboardData map[string]interface{}
		err = json.Unmarshal([]byte(clipboardContent), &clipboardData)
		if err != nil {
			log.Fatal("Failed to parse clipboard content as JSON: %v", err)
			return
		}

		// Check if the JSON has the expected structure
		mcpServers, ok := clipboardData["mcpServers"]
		if !ok {
			log.Fatal("Clipboard content does not contain 'mcpServers' key")
			return
		}

		// Convert to JSON string to reuse in MCPConfig
		mcpServersJSON, err := json.Marshal(mcpServers)
		if err != nil {
			log.Fatal("Failed to convert mcpServers to JSON: %v", err)
			return
		}

		// Parse mcpServers into our config structure
		var mcpConfig config.MCPConfig
		err = json.Unmarshal([]byte(fmt.Sprintf(`{"mcpServers": %s}`, string(mcpServersJSON))), &mcpConfig)
		if err != nil {
			log.Fatal("Failed to parse mcpServers config: %v", err)
			return
		}

		// Load existing config
		existingConfig, err := config.LoadMCPConfig()
		if err != nil {
			// If error is because the file doesn't exist, create a new config
			existingConfig = &config.MCPConfig{
				MCPServers: make(map[string]config.MCPServer),
			}
		}

		// Merge new servers into existing config
		for name, server := range mcpConfig.MCPServers {
			existingConfig.MCPServers[name] = server
			log.Info("Added MCP server: %s", name)
		}

		// Save the updated config
		err = config.SaveMCPConfig(existingConfig)
		if err != nil {
			log.Fatal("Failed to save config: %v", err)
			return
		}

		log.Info("Successfully loaded MCP configuration from clipboard")
	},
}

// getClipboard gets the content of the clipboard
func getClipboard() (string, error) {
	var cmd *exec.Cmd
	var out []byte
	var err error

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("pbpaste")
	case "linux":
		cmd = exec.Command("xclip", "-selection", "clipboard", "-o")
	case "windows":
		cmd = exec.Command("powershell.exe", "-command", "Get-Clipboard")
	default:
		return "", fmt.Errorf("unsupported platform")
	}

	out, err = cmd.Output()
	if err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			if len(out) > 0 {
				return string(out), nil
			}
			return "", fmt.Errorf("clipboard command failed: %v", err)
		}
		return "", fmt.Errorf("failed to execute clipboard command: %v", err)
	}

	return strings.TrimSpace(string(out)), nil
}

func init() {
	rootCmd.AddCommand(loadCmd)
}
