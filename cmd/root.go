package cmd

import (
	"fmt"
	"os"
	"regexp"

	"github.com/spf13/cobra"
)

var annotation string
var upgradeFrom string
var upgrade bool

// addUpgradeFlags registers the flags that control which upgrade-only images are included.
func addUpgradeFlags(cmd *cobra.Command) {
	cmd.Flags().BoolVar(&upgrade, "upgrade", false, "include all upgrade-only images")
	cmd.Flags().StringVar(&upgradeFrom, "upgrade-from", "", "include only the upgrade-only images needed to upgrade from this version (takes precedence over --upgrade)")
	// --no-upgrade is the default behavior since 1.0; the flag is kept so existing scripts don't break.
	var noUpgrade bool
	cmd.Flags().BoolVar(&noUpgrade, "no-upgrade", false, "no-op, upgrade-only images are skipped by default")
	cmd.Flags().MarkHidden("no-upgrade")
}

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
