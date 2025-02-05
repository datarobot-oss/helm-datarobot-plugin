package cmd

import (
	"fmt"
	"os"
	"regexp"

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

func SetVersionInfo(version, commit string) {
	re := regexp.MustCompile(`\d+\.\d+\.\d+`)
	// Find the first match of the pattern in the version string
	semver := re.FindString(version)
	rootCmd.Version = fmt.Sprintf("%s (Git SHA %s)", semver, commit)
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
