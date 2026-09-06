package cmd

import (
	"fmt"
	"io"
	"strings"

	"github.com/datarobot-oss/helm-datarobot-plugin/pkg/manifest"
	"github.com/spf13/cobra"
)

var filterResourcesCmd = &cobra.Command{
	Use:          "filter-resources",
	Short:        "Filter rendered Helm manifests by scope (stdin → stdout)",
	SilenceUsage: true,
	Long: strings.Replace(`
Read rendered Kubernetes manifests from stdin and write only the requested
partition to stdout. Use as a Helm post-renderer to strip cluster-scoped
resources before installing with limited privileges, or to extract only the
admin resources.

  --keep app   (default) write namespaced resources — for limited-privilege install
  --keep admin           write cluster-scoped resources only

Example:
'''sh
$ helm install datarobot ./datarobot-prime-11.10.88.tgz \
    --post-renderer ./helm-datarobot \
    --post-renderer-args filter-resources \
    --post-renderer-args --keep=app
'''

Note: --post-renderer-args requires Helm >= 3.5.
The binary is built at repo root via 'make build'.
`, "'", "`", -1),
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if frInput.Keep != "app" && frInput.Keep != "admin" {
			return fmt.Errorf("--keep must be 'app' or 'admin', got %q", frInput.Keep)
		}

		in, err := io.ReadAll(cmd.InOrStdin())
		if err != nil {
			return fmt.Errorf("read stdin: %w", err)
		}

		resources, err := manifest.ParseManifests(string(in))
		if err != nil {
			return fmt.Errorf("parse manifests: %w", err)
		}

		result := manifest.Classify(resources, frInput.ExtraAdminKinds)

		for _, w := range result.Warnings {
			cmd.PrintErrln("warning: " + w)
		}

		var kept []manifest.Resource
		if frInput.Keep == "admin" {
			kept = result.Admin
		} else {
			kept = result.App
		}

		if len(kept) == 0 {
			return nil
		}

		parts := make([]string, 0, len(kept))
		for _, r := range kept {
			parts = append(parts, r.RawYAML)
		}
		_, err = fmt.Fprintf(cmd.OutOrStdout(), "%s\n", strings.Join(parts, "\n---\n"))
		return err
	},
}

type filterResourcesInput struct {
	Keep            string
	ExtraAdminKinds []string
}

var frInput filterResourcesInput

func init() {
	rootCmd.AddCommand(filterResourcesCmd)
	filterResourcesCmd.Flags().StringVar(&frInput.Keep, "keep", "app", `Partition to keep: "app" (namespaced resources) or "admin" (cluster-scoped resources)`)
	filterResourcesCmd.Flags().StringSliceVar(&frInput.ExtraAdminKinds, "extra-admin-kinds", []string{}, "Resource kinds to treat as admin (cluster-scoped) even if namespaced")
}
