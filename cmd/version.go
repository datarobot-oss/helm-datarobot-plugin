package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Version of the application, set during build time if needed.
var version = "v1.0.0"

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "version",
	Long:  `version`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("version: %s\n", version)
	},
}

func init() {
	// Adding the version command to the root command
	rootCmd.AddCommand(versionCmd)
}
