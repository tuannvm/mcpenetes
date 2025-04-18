package cmd

import (
	"errors"
	"fmt"
	"net/url"

	"github.com/spf13/cobra"
	"github.com/tuannvm/mcpenetes/internal/config"
	"github.com/tuannvm/mcpenetes/internal/log"
)

// addRegistryCmd represents the add registry command
var addRegistryCmd = &cobra.Command{
	Use:   "registry [name] [url]",
	Short: "Adds a new MCP registry source.",
	Long:  `Adds a new named registry URL to the configuration file (config.yaml). This URL should point to a JSON index file listing available MCP versions.`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 2 {
			return errors.New("requires exactly two arguments: registry name and URL")
		}
		// Basic URL validation
		_, err := url.ParseRequestURI(args[1])
		if err != nil {
			// Use fmt.Errorf here as Cobra expects an error type
			return fmt.Errorf("invalid URL format: %w", err)
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		registryName := args[0]
		registryURL := args[1]

		log.Info("Adding registry: %s -> %s", registryName, registryURL)

		cfg, err := config.LoadConfig()
		if err != nil {
			log.Fatal("Error loading config: %v", err)
		}

		// Check if registry name already exists
		for _, reg := range cfg.Registries {
			if reg.Name == registryName {
				log.Fatal("Registry with name '%s' already exists.", registryName)
			}
		}

		// Add the new registry
		newRegistry := config.Registry{
			Name: registryName,
			URL:  registryURL,
		}
		cfg.Registries = append(cfg.Registries, newRegistry)

		// Save the updated config
		if err := config.SaveConfig(cfg); err != nil {
			log.Fatal("Error saving config: %v", err)
		}

		log.Success("Successfully added registry '%s'.", registryName)
	},
}

func init() {
	addCmd.AddCommand(addRegistryCmd)

	// You can add flags specific to this command here if needed
}
