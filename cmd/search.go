package cmd

import (
	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"
	"github.com/tuannvm/mcpenetes/internal/config"
	"github.com/tuannvm/mcpenetes/internal/log"
	"github.com/tuannvm/mcpenetes/internal/registry"
	"time"
	"github.com/briandowns/spinner"
)

// searchCmd represents the search command
var searchCmd = &cobra.Command{
	Use:   "search",
	Short: "Interactive fuzzy search for MCP versions",
	Long:  `Provides an interactive fuzzy search interface to find and select MCP versions from configured registries.`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Info("Starting interactive search...")

		cfg, err := config.LoadConfig()
		if err != nil {
			log.Fatal("Error loading config: %v", err)
		}

		if len(cfg.Registries) == 0 {
			log.Warn("No registries configured. Use 'mcpetes add registry <n> <url>' to add one.")
			return
		}

		// Start spinner
		s := spinner.New(spinner.CharSets[9], 100*time.Millisecond)
		s.Suffix = " Fetching available MCPs..."
		s.Start()

		var mcps []string
		for _, reg := range cfg.Registries {
			versions, err := registry.FetchMCPList(reg.URL)
			if err != nil {
				log.Warn("Error fetching from registry %s: %v", reg.URL, err)
				continue
			}
			mcps = append(mcps, versions...)
		}

		s.Stop()

		if len(mcps) == 0 {
			log.Warn("No MCP versions found in any registry")
			return
		}

		var selectedMCP string
		prompt := &survey.Select{
			Message: "Select MCP version:",
			Options: mcps,
		}
		
		err = survey.AskOne(prompt, &selectedMCP)
		if err != nil {
			log.Fatal("Error during selection: %v", err)
			return
		}

		log.Info("Selected MCP: %s", selectedMCP)

		// Use the selected MCP
		useCmd.Run(cmd, []string{selectedMCP})
	},
}

func init() {
	rootCmd.AddCommand(searchCmd)
}
