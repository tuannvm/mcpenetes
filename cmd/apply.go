package cmd

import (
	"fmt"
	"os"

	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"
	"github.com/tuannvm/mcpenetes/internal/config"
	"github.com/tuannvm/mcpenetes/internal/log"
	"github.com/tuannvm/mcpenetes/internal/translator"
	"github.com/tuannvm/mcpenetes/internal/util"
)

// applyCmd represents the apply command (renamed from reload)
var applyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Applies MCP configuration to all clients",
	Long: `Applies the MCP configuration to all compatible clients by:

1. Loading MCP server configurations from mcp.json
2. Automatically detecting installed MCP-compatible clients
3. Converting the configuration to formats compatible with each client:
   - Claude Desktop
   - Windsurf
   - Cursor
   - Visual Studio Code
4. Backing up existing configuration files before overwriting
5. Writing the new converted configuration for each client

This command requires confirmation before proceeding.`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Info("Preparing to apply MCP configuration...")

		// 1. Load configurations
		cfg, err := config.LoadConfig()
		if err != nil {
			log.Fatal("Error loading config.yaml: %v", err)
		}

		mcpCfg, err := config.LoadMCPConfig()
		if err != nil {
			log.Fatal("Error loading mcp.json: %v", err)
		}

		// Get the list of available servers from mcp.json
		if len(mcpCfg.MCPServers) == 0 {
			log.Fatal("No MCP servers found in mcp.json. Please add a server configuration first.")
		}

		// Check if clients are defined in config
		if len(cfg.Clients) == 0 {
			log.Info("No clients defined in config.yaml. Detecting installed clients...")

			// Auto-detect installed clients
			detectedClients, err := util.DetectMCPClients()
			if err != nil {
				log.Warn("Error detecting clients: %v", err)
			}

			if len(detectedClients) == 0 {
				log.Warn("No MCP-compatible clients detected on this system.")
				return
			}

			// Use the detected clients
			cfg.Clients = detectedClients
			log.Success("Detected %d client(s) on your system!", len(detectedClients))
		}

		if len(cfg.Clients) == 0 {
			log.Warn("No clients found to apply configuration to.")
			return
		}

		// Create a list of client names for selection
		var clientNames []string
		for name := range cfg.Clients {
			clientNames = append(clientNames, name)
		}
		clientNames = append(clientNames, "ALL") // Add option to select all clients

		// Let user choose which clients to apply to
		var selectedClients []string
		clientPrompt := &survey.MultiSelect{
			Message: "Select clients to apply MCP configuration to:",
			Options: clientNames,
			Default: []string{"ALL"}, // Default to ALL
		}
		
		// Use AskOne without a custom transformer (simpler approach)
		err = survey.AskOne(clientPrompt, &selectedClients, survey.WithValidator(survey.Required))
		if err != nil {
			log.Fatal("Error during client selection: %v", err)
		}
		
		// Process selections
		applyToAllClients := false
		for _, c := range selectedClients {
			if c == "ALL" {
				applyToAllClients = true
				break
			}
		}
		
		// Create a filtered client map
		selectedClientMap := make(map[string]config.Client)
		if applyToAllClients {
			selectedClientMap = cfg.Clients // Use all clients
		} else {
			// Only include selected clients
			for _, name := range selectedClients {
				if client, ok := cfg.Clients[name]; ok {
					selectedClientMap[name] = client
				}
			}
		}
		
		if len(selectedClientMap) == 0 {
			log.Warn("No clients selected. Nothing to apply.")
			return
		}

		// Generate client list for display
		clientList := ""
		for clientName := range selectedClientMap {
			clientList += fmt.Sprintf("  - %s\n", clientName)
		}

		// Generate server list for display
		serverList := ""
		for serverName := range mcpCfg.MCPServers {
			serverList += fmt.Sprintf("  - %s\n", serverName)
		}

		// Ask for confirmation
		confirmMessage := fmt.Sprintf("This will apply ALL MCP server configurations to the following clients:\n%s\nThe following MCP servers will be applied:\n%s\nBackups will be created. Do you want to continue?", clientList, serverList)
		var confirm bool
		prompt := &survey.Confirm{
			Message: confirmMessage,
			Default: false, // Safer default - user must explicitly choose yes
		}

		err = survey.AskOne(prompt, &confirm)
		if err != nil {
			log.Fatal("Error during confirmation: %v", err)
		}

		if !confirm {
			log.Info("Operation cancelled by user.")
			return
		}

		// Create Translator
		trans := translator.NewTranslator(cfg, mcpCfg)

		// Process all clients and all servers
		log.Info("Processing clients and servers...")
		clientSuccessCount := 0
		clientFailureCount := 0
		totalOperations := 0

		// For each selected client
		for clientName, clientConf := range selectedClientMap {
			log.Printf(log.InfoColor, "- Processing client: %s\n", clientName)

			// Backup client config once before making any changes
			backupPath, err := trans.BackupClientConfig(clientName, clientConf)
			if err != nil {
				log.Error("  Error backing up config for %s: %v", clientName, err)
				clientFailureCount++
				continue // Skip this client if backup failed
			}
			log.Success("  Created backup at: %s", backupPath)

			clientSuccess := true

			// Apply each server configuration to this client
			for serverName, serverConf := range mcpCfg.MCPServers {
				log.Printf(log.InfoColor, "  - Applying server: %s\n", serverName)

				// Translate and Apply
				err = trans.TranslateAndApply(clientName, clientConf, serverConf)
				if err != nil {
					log.Error("    Error applying server %s to client %s: %v", serverName, clientName, err)
					clientSuccess = false
				} else {
					log.Success("    Successfully applied server %s to client %s", serverName, clientName)
					totalOperations++
				}
			}

			if clientSuccess {
				clientSuccessCount++
			} else {
				clientFailureCount++
			}
		}

		log.Info("\nApply operation finished.")
		log.Success("Successfully applied %d server configurations across %d clients.", totalOperations, clientSuccessCount)
		if clientFailureCount > 0 {
			log.Error("Failed to apply to %d clients.", clientFailureCount)
			os.Exit(1) // Exit with error if any client failed
		}
	},
}

func init() {
	rootCmd.AddCommand(applyCmd)
}
