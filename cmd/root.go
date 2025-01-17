package cmd

import (
	"bytes"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/mattn/go-shellwords"
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

func executeCommand(root *cobra.Command, cmd string) (output string, err error) {
	buf := new(bytes.Buffer)

	args, err := shellwords.Parse(cmd)
	if err != nil {
		return "", err
	}

	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs(args)

	err = root.Execute()
	return strings.TrimSpace(buf.String()), err
}
