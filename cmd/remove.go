package cmd

import (
	"github.com/spf13/cobra"
)

// removeCmd represents the remove command
var removeCmd = &cobra.Command{
	Use:   "remove",
	Short: "Remove resources like registries.",
	Long:  `Parent command for removing different types of resources managed by mcpetes.`,
	Aliases: []string{"rm"}, // Add 'rm' as an alias
	// Run: func(cmd *cobra.Command, args []string) { 
	// 	 cmd.Help()
	// },
}

func init() {
	rootCmd.AddCommand(removeCmd)

	// Here you will define your flags and configuration settings.
}
