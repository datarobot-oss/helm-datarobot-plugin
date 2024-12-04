package cmd

import (
	"github.com/spf13/cobra"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "version",
	Long:  `version`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Print(rootCmd.Version)
	},
}

func init() {
	// Adding the version command to the root command
	rootCmd.AddCommand(versionCmd)
}
