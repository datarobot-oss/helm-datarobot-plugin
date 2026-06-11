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
	resetSubCommandFlagValues(root) // See: https://github.com/spf13/cobra/issues/1488
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs(args)

	err = root.Execute()
	return strings.TrimSpace(buf.String()), err
}

// From: https://github.com/golang/debug/pull/8/files
// resetSubCommandFlagValues resets all changed flags to their default values.
// For pflag.SliceValue flags (stringSlice, stringArray, etc.) the DefValue is
// a bracketed CSV like "[a,b,c]"; calling f.Value.Set on it APPENDS rather than
// replaces, so we use Replace instead to fully overwrite the current slice.
func resetSubCommandFlagValues(root *cobra.Command) {
	for _, c := range root.Commands() {
		c.Flags().VisitAll(func(f *pflag.Flag) {
			if f.Changed {
				if sv, ok := f.Value.(pflag.SliceValue); ok {
					// Parse bracketed CSV DefValue: "[a,b,c]" → ["a","b","c"]
					// Empty default "[]" → nil (Replace with nil resets to empty).
					dv := strings.TrimSpace(f.DefValue)
					inner := strings.TrimSuffix(strings.TrimPrefix(dv, "["), "]")
					var parsed []string
					if inner != "" {
						for _, s := range strings.Split(inner, ",") {
							parsed = append(parsed, strings.TrimSpace(s))
						}
					}
					sv.Replace(parsed) //nolint:errcheck
				} else {
					f.Value.Set(f.DefValue) //nolint:errcheck
				}
				f.Changed = false
			}
		})
		resetSubCommandFlagValues(c)
	}
}
