package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "mcpetes",
	Short: "A CLI tool to manage multiple MCP endpoint configurations.",
	Long: `mcpetes helps you switch between different Model Context Protocol (MCP)
server configurations defined in a central mcp.json file or fetched from registries.
It can update configuration files for various clients (like VS Code extensions)
based on the selected MCP server.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Initialize logging level based on flags
		// verbose, _ := cmd.Flags().GetBool("verbose") // Flags can be checked in specific commands if needed
		// debug, _ := cmd.Flags().GetBool("debug")
		// log.Init(verbose, debug) // log package does not have Init function
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		// Cobra prints the error, but we might want to log it too or exit differently
		// log.Fatal("Command execution failed: %v", err) // Avoid double printing
		os.Exit(1)
	}
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Enable verbose output")
	// rootCmd.PersistentFlags().Bool("debug", false, "Enable debug output (more verbose)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	// rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	
	// Disable the auto-generated completion command as requested
	rootCmd.CompletionOptions.DisableDefaultCmd = true
}
