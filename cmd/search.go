package cmd

import (
	"errors"
	"fmt"
	"os/exec"
	"runtime"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/briandowns/spinner"
	"github.com/spf13/cobra"
	"github.com/tuannvm/mcpenetes/internal/config"
	"github.com/tuannvm/mcpenetes/internal/log"
	"github.com/tuannvm/mcpenetes/internal/registry"
)

// ServerInfo represents information about an MCP server
type ServerInfo struct {
	Name          string
	Description   string
	RepositoryURL string
}

// searchCmd represents the search command
var searchCmd = &cobra.Command{
	Use:   "search [server-id]",
	Short: "Interactive fuzzy search for MCP versions and apply them",
	Long: `Provides an interactive fuzzy search interface to find and select MCP versions from configured registries.
If a server ID is provided as an argument, it will directly use that server without prompting.
After selection, adds the server to the 'mcps' list in config.yaml.
This determines which server configuration will be used by the 'reload' command.`,
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
		cfg, err := config.LoadConfig()
		if err != nil {
			log.Fatal("Error loading config: %v", err)
		}

		// Direct server selection if argument provided
		if len(args) == 1 {
			serverID := args[0]
			log.Info("Using provided server ID: %s", serverID)

			// Add the selected server to the MCPs list in config
			cfg.MCPs = append(cfg.MCPs, serverID)

			// Save the updated config
			if err := config.SaveConfig(cfg); err != nil {
				log.Fatal("Error saving config: %v", err)
			}

			log.Success("Successfully added MCP '%s' to the list. Run 'mcpetes reload' to apply.", serverID)
			return
		}

		// Interactive selection mode
		log.Info("Starting interactive search...")

		if len(cfg.Registries) == 0 {
			log.Warn("No registries configured. Use 'mcpetes add registry <n> <url>' to add one.")
			return
		}

		// Start spinner
		s := spinner.New(spinner.CharSets[9], 100*time.Millisecond)
		s.Suffix = " Fetching available MCPs..."
		s.Start()

		var serverInfos []ServerInfo
		var displayOptions []string

		for _, reg := range cfg.Registries {
			servers, err := registry.FetchMCPServers(reg.URL)
			if err != nil {
				log.Warn("Error fetching from registry %s: %v", reg.URL, err)
				continue
			}

			for _, server := range servers {
				info := ServerInfo{
					Name:          server.Name,
					Description:   server.Description,
					RepositoryURL: server.RepositoryURL,
				}
				serverInfos = append(serverInfos, info)

				// Create display option string
				displayText := server.Name
				if server.Description != "" {
					displayText = fmt.Sprintf("%s: %s", server.Name, server.Description)
				}
				displayOptions = append(displayOptions, displayText)
			}
		}

		s.Stop()

		if len(serverInfos) == 0 {
			log.Warn("No MCP servers found in any registry")
			return
		}

		var selectedOption string
		prompt := &survey.Select{
			Message: "Select MCP server:",
			Options: displayOptions,
		}

		err = survey.AskOne(prompt, &selectedOption)
		if err != nil {
			log.Fatal("Error during selection: %v", err)
			return
		}

		// Find the index of the selected option
		selectedIndex := -1
		for i, opt := range displayOptions {
			if opt == selectedOption {
				selectedIndex = i
				break
			}
		}

		if selectedIndex == -1 {
			log.Fatal("Selected option not found in options list")
			return
		}

		selectedServer := serverInfos[selectedIndex]
		log.Info("Selected MCP: %s", selectedServer.Name)

		// If repository URL is available, ask if user wants to open it
		if selectedServer.RepositoryURL != "" {
			var openRepo bool
			confirmPrompt := &survey.Confirm{
				Message: fmt.Sprintf("Would you like to open the repository URL (%s) in your browser?", selectedServer.RepositoryURL),
				Default: true,
			}

			err = survey.AskOne(confirmPrompt, &openRepo)
			if err != nil {
				log.Warn("Error during confirmation: %v", err)
			}

			if openRepo {
				err := openBrowser(selectedServer.RepositoryURL)
				if err != nil {
					log.Warn("Failed to open browser: %v", err)
				} else {
					log.Info("Opened repository URL in browser")
					log.Info("After copying the configuration, run 'mcpenetes load' to load it from clipboard")
					return
				}
			}
		}

		// If user didn't open repo or there was an error, use the selected MCP
		serverID := selectedServer.Name
		log.Info("Using server ID: %s", serverID)

		cfg, err = config.LoadConfig()
		if err != nil {
			log.Fatal("Error loading config: %v", err)
		}

		// Add the selected server to the MCPs list in config
		cfg.MCPs = append(cfg.MCPs, serverID)

		// Save the updated config
		if err := config.SaveConfig(cfg); err != nil {
			log.Fatal("Error saving config: %v", err)
		}

		log.Success("Successfully added MCP '%s' to the list. Run 'mcpetes reload' to apply.", serverID)
	},
}

// openBrowser opens the specified URL in the default browser
func openBrowser(url string) error {
	var err error

	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}

	return err
}

func init() {
	rootCmd.AddCommand(searchCmd)
}
