package cmd

import (
	"bytes"
	"strings"

	"github.com/mattn/go-shellwords"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func executeCommand(root *cobra.Command, cmd string) (output string, err error) {
	buf := new(bytes.Buffer)

	args, err := shellwords.Parse(cmd)
	if err != nil {
		return "", err
	}
	root.Flags().VisitAll(func(flag *pflag.Flag) {
		flag.Value.Set(flag.DefValue)
		flag.Changed = false // Mark the flag as not changed
	})
	resetSubCommandFlagValues(rootCmd) // See: https://github.com/spf13/cobra/issues/1488
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs(args)

	err = root.Execute()
	return strings.TrimSpace(buf.String()), err
}

// From: https://github.com/golang/debug/pull/8/files
func resetSubCommandFlagValues(root *cobra.Command) {
	for _, c := range root.Commands() {
		c.Flags().VisitAll(func(flag *pflag.Flag) {
			flag.Value.Set(flag.DefValue)
			flag.Changed = false // Mark the flag as not changed
		})
		resetSubCommandFlagValues(c)
	}
}
