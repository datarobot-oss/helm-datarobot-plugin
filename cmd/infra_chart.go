package cmd

import (
	"fmt"
	"strings"

	"github.com/datarobot-oss/helm-datarobot-plugin/pkg/infrachart"
	"github.com/datarobot-oss/helm-datarobot-plugin/pkg/manifest"
	"github.com/datarobot-oss/helm-datarobot-plugin/pkg/render_helper"
	"github.com/spf13/cobra"
)

type infraChartInput struct {
	Namespace       string
	ReleaseName     string
	ValueFiles      []string
	Values          []string
	KubeVersion     string
	APIVersions     []string
	Output          string
	KeepCRDs        bool
	Debug           bool
	ExtraAdminKinds []string
	// Task 5 adds: PipelineSA, ClusterReadKinds, CRDAggregation
}

var ic infraChartInput

var infraChartCmd = &cobra.Command{
	Use:          "infra-chart <prime-chart.tgz>",
	Short:        "extract all cluster-scoped resources from a chart into a standalone datarobot-infra chart",
	SilenceUsage: true,
	Long: strings.Replace(`
Render datarobot-prime and extract all cluster-scoped resources (CRDs,
ClusterRoles, ClusterRoleBindings, webhooks, PriorityClass, GatewayClass,
APIService, ...) into a standalone datarobot-infra chart. The source chart is
read-only and never modified.

Example:
'''sh
$ helm datarobot infra-chart datarobot-prime.tgz -o datarobot-infra.tgz
'''`, "'", "`", -1),
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		chartPath := args[0]

		rendered, err := render_helper.RenderChart(chartPath, ic.ValueFiles, ic.Values, &render_helper.RenderOptions{
			Namespace:    ic.Namespace,
			ReleaseName:  ic.ReleaseName,
			KubeVersion:  ic.KubeVersion,
			IncludeCRDs:  true,
			IncludeHooks: true,
			APIVersions:  ic.APIVersions,
		})
		if err != nil {
			return fmt.Errorf("failed to render chart: %w", err)
		}

		resources, err := manifest.ParseManifests(rendered)
		if err != nil {
			return fmt.Errorf("failed to parse manifests: %w", err)
		}

		result := manifest.Classify(resources, ic.ExtraAdminKinds)
		for _, w := range result.Warnings {
			cmd.PrintErrln("warning: " + w)
		}

		if len(result.Admin) == 0 {
			return fmt.Errorf("no cluster-scoped resources found in rendered output — check values/flags")
		}

		src, err := infrachart.ReadSourceChartMeta(chartPath)
		if err != nil {
			return fmt.Errorf("failed to read source chart metadata: %w", err)
		}
		version := src.Version
		if version == "" {
			cmd.PrintErrln("warning: source chart has no version; stamping 0.0.0")
			version = "0.0.0"
		}

		outputPath := ic.Output
		if outputPath == "" {
			outputPath = fmt.Sprintf("./datarobot-infra-%s.tgz", version)
		}

		chartOpts := infrachart.ChartOptions{
			Name:       "datarobot-infra",
			Version:    version,
			AppVersion: src.AppVersion,
			SourceName: src.Name,
			KeepCRDs:   ic.KeepCRDs,
		}

		builtChart, err := infrachart.BuildChart(result.Admin, chartOpts)
		if err != nil {
			return fmt.Errorf("failed to build infra chart: %w", err)
		}
		if err := infrachart.PackageChart(builtChart, outputPath); err != nil {
			return fmt.Errorf("failed to package chart: %w", err)
		}

		cmd.Printf("Extracted %d cluster-scoped resources (skipped %d namespaced): %s\n",
			len(result.Admin), len(result.App), manifest.Summary(result.Admin))
		cmd.Printf("Infra chart written to: %s\n", outputPath)

		if ic.Debug {
			cmd.Println("\nResources:")
			for _, r := range result.Admin {
				cmd.Printf("  %s/%s\n", r.Kind, r.Name)
			}
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(infraChartCmd)
	infraChartCmd.Flags().StringVar(&ic.Namespace, "namespace", "datarobot", "render namespace (.Release.Namespace)")
	infraChartCmd.Flags().StringVar(&ic.ReleaseName, "release-name", "datarobot", ".Release.Name")
	infraChartCmd.Flags().StringSliceVarP(&ic.ValueFiles, "values", "f", []string{}, "specify values in a YAML file (can specify multiple)")
	infraChartCmd.Flags().StringArrayVar(&ic.Values, "set", []string{}, "set values on the command line (can specify multiple)")
	infraChartCmd.Flags().StringVar(&ic.KubeVersion, "kube-version", "v1.32.0", "Helm template KubeVersion")
	infraChartCmd.Flags().StringSliceVar(&ic.APIVersions, "api-versions", []string{}, "extra API versions for rendering (can specify multiple)")
	infraChartCmd.Flags().StringVarP(&ic.Output, "output", "o", "", "output .tgz path (default ./datarobot-infra-<srcVersion>.tgz)")
	infraChartCmd.Flags().BoolVar(&ic.KeepCRDs, "keep-crds", true, "add helm.sh/resource-policy: keep annotation to CRDs")
	infraChartCmd.Flags().BoolVarP(&ic.Debug, "debug", "d", false, "verbose per-resource listing")
	infraChartCmd.Flags().StringSliceVar(&ic.ExtraAdminKinds, "extra-admin-kinds", []string{}, "resource kinds to force into the infra chart even if namespaced (e.g. Role,RoleBinding,ServiceAccount)")
}
