package cmd

import (
	// "fmt"
	// "os"
	"sync"
	"time"

	"github.com/briandowns/spinner" // Added spinner
	"github.com/spf13/cobra"
	"github.com/tuannvm/mcpenetes/internal/config"
	"github.com/tuannvm/mcpenetes/internal/log"
	"github.com/tuannvm/mcpenetes/internal/registry"
)

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "Lists available MCP versions from configured registries.",
	Long:  `Fetches and displays the list of available Minecraft server (MCP) versions from all registries defined in config.yaml.`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Info("Executing list command...")

		cfg, err := config.LoadConfig()
		if err != nil {
			log.Fatal("Error loading config: %v", err)
		}

		if len(cfg.Registries) == 0 {
			log.Warn("No registries configured. Use 'mcpetes add registry <name> <url>' to add one.")
			return
		}

		log.Info("Fetching MCPs from registries...")

		// Start spinner
		s := spinner.New(spinner.CharSets[9], 100*time.Millisecond) // Use dot spinner
		s.Suffix = " Fetching..."
		s.Start()

		// Use a map to store results, keyed by registry name
		mcpLists := make(map[string][]string)
		var mu sync.Mutex // Mutex to protect concurrent map writes
		var wg sync.WaitGroup // WaitGroup to wait for all fetches to complete

		for _, reg := range cfg.Registries {
			wg.Add(1)
			go func(r config.Registry) { // Fetch concurrently
				defer wg.Done()
				versions, err := registry.FetchMCPList(r.URL) // FetchMCPList logs cache status
				mu.Lock()
				defer mu.Unlock()
				if err != nil {
					// Log warning, but don't print during spinner
					mcpLists[r.Name] = []string{"<error>"} // Indicate error
				} else {
					mcpLists[r.Name] = versions
				}
			}(reg)
		}

		wg.Wait() // Wait for all goroutines to finish
		s.Stop() // Stop spinner

		// Now print any errors that occurred during fetch
		for name, versions := range mcpLists {
			if len(versions) > 0 && versions[0] == "<error>" {
				// Find the original URL to include in the error message
				var url string
				for _, reg := range cfg.Registries {
					if reg.Name == name {
						url = reg.URL
						break
					}
				}
				log.Warn("  Error fetching from registry '%s' (%s)", name, url) // Actual error details are logged by FetchMCPList/cache
			}
		}

		log.Info("\nAvailable MCPs:")
		foundAny := false
		for name, versions := range mcpLists {
			if len(versions) > 0 && versions[0] == "<error>" {
				log.Printf(log.ErrorColor, "- %s: Error fetching versions\n", name)
				continue
			}
			if len(versions) == 0 {
				log.Printf(log.WarnColor, "- %s: No versions found\n", name)
				continue
			}
			foundAny = true
			log.Printf(log.InfoColor, "- %s:\n", name)
			for _, version := range versions {
				log.Printf(log.DetailColor, "    - %s\n", version)
			}
		}

		if !foundAny {
			log.Warn("  No MCP versions found in any registry (or errors occurred).")
		}
	},
}

func init() {
	rootCmd.AddCommand(listCmd)

	// Here you will define your flags and configuration settings.

	// Example local flag:
	// listCmd.Flags().BoolP("verbose", "v", false, "Show verbose output")
}
