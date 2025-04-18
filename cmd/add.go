package cmd

import (
	"github.com/spf13/cobra"
)

// addCmd represents the add command
var addCmd = &cobra.Command{
	Use:   "add",
	Short: "Add resources like registries or servers.",
	Long:  `Parent command for adding different types of resources managed by mcpetes.`,
	// Run: func(cmd *cobra.Command, args []string) { 
	// 	 // If called without subcommand, maybe show help?
	// 	 cmd.Help()
	// },
}

func init() {
	rootCmd.AddCommand(addCmd)

	// Here you will define your flags and configuration settings.
}
