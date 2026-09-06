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
func resetSubCommandFlagValues(root *cobra.Command) {
	for _, c := range root.Commands() {
		c.Flags().VisitAll(func(f *pflag.Flag) {
			if !f.Changed {
				return
			}
			// Slice/array flags (e.g. --set) must be Replace()d back to their
			// default: their Set() appends once the value is marked changed, so
			// Set(DefValue) would push the literal "[]" onto the slice and the
			// next test's --set data would fail to parse.
			if sv, ok := f.Value.(pflag.SliceValue); ok {
				sv.Replace(parseSliceDefault(f.DefValue))
			} else {
				f.Value.Set(f.DefValue)
			}
			f.Changed = false
		})
		resetSubCommandFlagValues(c)
	}
}

// parseSliceDefault turns pflag's "[a,b,c]" DefValue rendering back into the
// slice of defaults, so resetting a changed slice flag restores its real
// default (empty for --set, non-empty for e.g. --cluster-read-kinds).
func parseSliceDefault(def string) []string {
	def = strings.TrimSpace(def)
	def = strings.TrimPrefix(def, "[")
	def = strings.TrimSuffix(def, "]")
	if def == "" {
		return []string{}
	}
	return strings.Split(def, ",")
}
