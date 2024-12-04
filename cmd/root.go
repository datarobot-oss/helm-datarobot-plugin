package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var annotation string

var rootCmd = &cobra.Command{
	Use:   "helm-datarobot",
	Short: "datarobot helm plugin",
	Long:  `datarobot helm plugin`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Use --help for more information.")
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
