package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/tuannvm/mcpenetes/internal/config"
)

// removeRegistryCmd represents the remove registry command
var removeRegistryCmd = &cobra.Command{
	Use:   "registry [name]",
	Short: "Removes an MCP registry source.",
	Long:  `Removes a named registry from the configuration file (config.yaml).`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			return errors.New("requires exactly one argument: the name of the registry to remove")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		registryName := args[0]

		fmt.Printf("Removing registry: %s\n", registryName)

		cfg, err := config.LoadConfig()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
			os.Exit(1)
		}

		foundIndex := -1
		for i, reg := range cfg.Registries {
			if reg.Name == registryName {
				foundIndex = i
				break
			}
		}

		if foundIndex == -1 {
			fmt.Fprintf(os.Stderr, "Error: Registry with name '%s' not found.\n", registryName)
			os.Exit(1)
		}

		// Remove the registry by creating a new slice without it
		cfg.Registries = append(cfg.Registries[:foundIndex], cfg.Registries[foundIndex+1:]...)

		// Save the updated config
		if err := config.SaveConfig(cfg); err != nil {
			fmt.Fprintf(os.Stderr, "Error saving config: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Successfully removed registry '%s'.\n", registryName)
	},
}

func init() {
	removeCmd.AddCommand(removeRegistryCmd)
}
