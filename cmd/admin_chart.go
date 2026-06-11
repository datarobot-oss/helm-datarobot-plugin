package cmd

import (
	"fmt"
	"strings"

	"github.com/datarobot-oss/helm-datarobot-plugin/pkg/adminchart"
	"github.com/datarobot-oss/helm-datarobot-plugin/pkg/manifest"
	"github.com/datarobot-oss/helm-datarobot-plugin/pkg/render_helper"
	"github.com/spf13/cobra"
)

var adminChartCmd = &cobra.Command{
	Use:          "admin-chart",
	Short:        "Generate a helm chart containing all cluster-scoped resources",
	SilenceUsage: true,
	Long: strings.Replace(`
Extract all cluster-scoped resources (CRDs, ClusterRoles, ClusterRoleBindings,
Webhooks, etc.) from a DataRobot Helm chart and package them as a standalone
installable Helm chart.

Example:
'''sh
$ helm datarobot admin-chart ./datarobot-prime-11.10.88.tgz \
    --namespace dr \
    --values my-values.yaml \
    --output ./datarobot-admin-11.10.88.tgz
'''`, "'", "`", -1),
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		chartPath := args[0]

		if acInput.Namespace == "" {
			return fmt.Errorf("--namespace is required")
		}

		if acInput.Output == "" {
			return fmt.Errorf("--output is required")
		}

		renderOpts := &render_helper.RenderOptions{
			Namespace:   acInput.Namespace,
			ReleaseName: acInput.ReleaseName,
			KubeVersion: acInput.KubeVersion,
			IncludeCRDs: true,
			APIVersions: acInput.APIVersions,
		}

		if acInput.OpenShift {
			renderOpts.APIVersions = append(renderOpts.APIVersions,
				"security.openshift.io/v1",
				"route.openshift.io/v1",
			)
		}

		rendered, err := render_helper.RenderChart(chartPath, acInput.ValueFiles, acInput.Values, renderOpts)
		if err != nil {
			return fmt.Errorf("failed to render chart: %w", err)
		}

		resources, err := manifest.ParseManifests(rendered)
		if err != nil {
			return fmt.Errorf("failed to parse manifests: %w", err)
		}

		clusterScoped, _ := manifest.FilterClusterScoped(resources)

		if len(clusterScoped) == 0 {
			return fmt.Errorf("no cluster-scoped resources found in rendered output")
		}

		sourceMeta, err := adminchart.ReadSourceChartMeta(chartPath)
		if err != nil {
			return fmt.Errorf("failed to read source chart metadata: %w", err)
		}

		chartOpts := adminchart.ChartOptions{
			Name:       "datarobot-admin",
			Version:    sourceMeta.Version,
			AppVersion: sourceMeta.AppVersion,
		}

		adminChart, err := adminchart.BuildChart(clusterScoped, chartOpts)
		if err != nil {
			return fmt.Errorf("failed to build admin chart: %w", err)
		}

		if err := adminchart.PackageChart(adminChart, acInput.Output); err != nil {
			return fmt.Errorf("failed to package chart: %w", err)
		}

		summary := manifest.Summary(clusterScoped)
		cmd.Printf("Extracted %d cluster-scoped resources: %s\n", len(clusterScoped), summary)
		cmd.Printf("Admin chart written to: %s\n", acInput.Output)

		if acInput.Debug {
			cmd.Println("\nResources:")
			for _, r := range clusterScoped {
				cmd.Printf("  %s/%s\n", r.Kind, r.Name)
			}
		}

		return nil
	},
}

type adminChartInput struct {
	Namespace   string
	ReleaseName string
	KubeVersion string
	Output      string
	ValueFiles  []string
	Values      []string
	APIVersions []string
	OpenShift   bool
	Debug       bool
}

var acInput adminChartInput

func init() {
	rootCmd.AddCommand(adminChartCmd)
	adminChartCmd.Flags().StringVar(&acInput.Namespace, "namespace", "", "Kubernetes namespace (required)")
	adminChartCmd.Flags().StringVar(&acInput.ReleaseName, "release-name", "datarobot", "Helm release name")
	adminChartCmd.Flags().StringVar(&acInput.KubeVersion, "kube-version", "v1.32.0", "Kubernetes version for template rendering")
	adminChartCmd.Flags().StringVarP(&acInput.Output, "output", "o", "", "Output path for the admin chart .tgz (required)")
	adminChartCmd.Flags().StringSliceVarP(&acInput.ValueFiles, "values", "f", []string{}, "Specify values in a YAML file (can specify multiple)")
	adminChartCmd.Flags().StringArrayVar(&acInput.Values, "set", []string{}, "Set values on the command line (can specify multiple)")
	adminChartCmd.Flags().StringSliceVar(&acInput.APIVersions, "api-versions", []string{}, "Additional API versions for template rendering")
	adminChartCmd.Flags().BoolVar(&acInput.OpenShift, "openshift", false, "Include OpenShift API versions (security.openshift.io/v1, route.openshift.io/v1)")
	adminChartCmd.Flags().BoolVarP(&acInput.Debug, "debug", "d", false, "Print detailed resource listing")
}
