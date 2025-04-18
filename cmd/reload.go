package cmd

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/tuannvm/mcpenetes/internal/config"
	"github.com/tuannvm/mcpenetes/internal/log"
	"github.com/tuannvm/mcpenetes/internal/translator"
)

// reloadCmd represents the reload command
var reloadCmd = &cobra.Command{
	Use:   "reload",
	Short: "Applies the selected MCP configuration to all clients.",
	Long: `Reloads the configuration by:
1. Reading the selected server ID from config.yaml.
2. Finding the corresponding server definition in mcp.json.
3. Backing up existing configuration files for each client.
4. Translating and writing the new configuration for each client.`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Info("Executing reload command...")

		// 1. Load configurations
		cfg, err := config.LoadConfig()
		if err != nil {
			log.Fatal("Error loading config.yaml: %v", err)
		}

		if cfg.SelectedMCP == "" {
			log.Fatal("No MCP server selected. Use 'mcpetes use <server-id>' first.")
		}

		mcpCfg, err := config.LoadMCPConfig()
		if err != nil {
			log.Fatal("Error loading mcp.json: %v", err)
		}

		// 2. Find the selected server configuration
		selectedServerConf, found := mcpCfg.MCPServers[cfg.SelectedMCP]
		if !found {
			log.Fatal("Selected MCP server '%s' not found in mcp.json.", cfg.SelectedMCP)
		}
		log.Info("Applying configuration for server: %s", cfg.SelectedMCP)

		// 3. Create Translator
		trans := translator.NewTranslator(cfg, mcpCfg)

		// 4. Iterate through clients, backup, and translate
		if len(cfg.Clients) == 0 {
			log.Warn("No clients defined in config.yaml. Nothing to reload.")
			return
		}

		log.Info("Processing clients:")
		successCount := 0
		failureCount := 0
		for clientName, clientConf := range cfg.Clients {
			log.Printf(log.InfoColor, "- Processing %s:\n", clientName)

			// Backup
			// BackupClientConfig now logs its own details/success
			_, err := trans.BackupClientConfig(clientName, clientConf)
			if err != nil {
				log.Error("  Error backing up config for %s: %v", clientName, err)
				failureCount++
				continue // Skip applying if backup failed?
			}

			// Translate and Apply
			// TranslateAndApply now logs its own details/success
			err = trans.TranslateAndApply(clientName, clientConf, selectedServerConf)
			if err != nil {
				log.Error("  Error applying config for %s: %v", clientName, err)
				failureCount++
				continue
			}
			successCount++
		}

		log.Info("\nReload finished.")
		log.Success("Successfully applied to %d clients.", successCount)
		if failureCount > 0 {
			log.Error("Failed to apply to %d clients.", failureCount)
			os.Exit(1) // Exit with error if any client failed
		}
	},
}

func init() {
	rootCmd.AddCommand(reloadCmd)
}
