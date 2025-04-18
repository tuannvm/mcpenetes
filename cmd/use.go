package cmd

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"sync"

	"github.com/AlecAivazis/survey/v2" // Added survey
	"github.com/spf13/cobra"
	"github.com/tuannvm/mcpenetes/internal/config"
	"github.com/tuannvm/mcpenetes/internal/registry" // Added registry
)

// useCmd represents the use command
var useCmd = &cobra.Command{
	Use:   "use [server-id]",
	Short: "Selects the active MCP server configuration, interactively if no ID is provided.",
	Long: `Sets the 'selected_mcp' value in config.yaml to the provided server ID.
If no server ID is provided as an argument, it fetches all available servers
(from registries and local mcp.json) and presents an interactive selection prompt.
This determines which server configuration from mcp.json will be used by the 'reload' command.`,
	Args: func(cmd *cobra.Command, args []string) error {
		// Allow 0 or 1 argument
		if len(args) > 1 {
			return errors.New("accepts at most one argument: the server ID to use")
		}
		if len(args) == 1 && args[0] == "" {
			return errors.New("server ID cannot be empty if provided")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		var serverID string

		cfg, err := config.LoadConfig()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
			os.Exit(1)
		}

		if len(args) == 1 {
			serverID = args[0]
			fmt.Printf("Using provided server ID: %s\n", serverID)
			// Optional: Validate provided serverID exists (see below)
		} else {
			// No argument provided, run interactive selection
			fmt.Println("Fetching available servers for selection...")
			choices, err := getAvailableServerChoices(cfg)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error getting server choices: %v\n", err)
				os.Exit(1)
			}

			if len(choices) == 0 {
				fmt.Println("No servers found in mcp.json or registries.")
				fmt.Println("Define servers in mcp.json or add registries using 'mcpetes add registry'.")
				return
			}

			prompt := &survey.Select{
				Message: "Choose an MCP server to use:",
				Options: choices,
				PageSize: 15, // Adjust as needed
			}
			err = survey.AskOne(prompt, &serverID)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error during selection: %v\n", err)
				os.Exit(1)
			}
		}

		if serverID == "" {
			fmt.Println("No server selected.")
			return
		}

		// --- Save the selected server ID --- 
		fmt.Printf("Setting active MCP server to: %s\n", serverID)

		// Update the selected MCP in the already loaded config
		cfg.SelectedMCP = serverID

		// Save the updated config
		if err := config.SaveConfig(cfg); err != nil {
			fmt.Fprintf(os.Stderr, "Error saving config: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Successfully set active MCP to '%s'. Run 'mcpetes reload' to apply.\n", serverID)
	},
}

// getAvailableServerChoices fetches servers from registries and mcp.json for interactive selection.
func getAvailableServerChoices(cfg *config.Config) ([]string, error) {
	mcpCfg, err := config.LoadMCPConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load mcp.json: %w", err)
	}

	// Use a map to avoid duplicates and store choices
	choicesMap := make(map[string]bool)

	// 1. Add servers defined locally in mcp.json
	for id := range mcpCfg.MCPServers {
		choicesMap[id] = true
	}

	// 2. Fetch servers/versions from registries (concurrently)
	if len(cfg.Registries) > 0 {
		var mu sync.Mutex
		var wg sync.WaitGroup
		registryResults := make(map[string][]string)

		for _, reg := range cfg.Registries {
			wg.Add(1)
			go func(r config.Registry) {
				defer wg.Done()
				// Note: FetchMCPList now uses cache internally
				versions, err := registry.FetchMCPList(r.URL)
				mu.Lock()
				defer mu.Unlock()
				if err != nil {
					// Log error but don't fail the whole process
					fmt.Fprintf(os.Stderr, "Warning: Error fetching from registry '%s': %v\n", r.Name, err)
				} else {
					registryResults[r.Name] = versions
				}
			}(reg)
		}
		wg.Wait()

		// Add fetched versions to choices map
		for _, versions := range registryResults {
			for _, v := range versions {
				choicesMap[v] = true
			}
		}
	}

	// Convert map keys to a sorted slice for consistent order
	choices := make([]string, 0, len(choicesMap))
	for choice := range choicesMap {
		choices = append(choices, choice)
	}
	sort.Strings(choices)

	return choices, nil
}

func init() {
	rootCmd.AddCommand(useCmd)
}
