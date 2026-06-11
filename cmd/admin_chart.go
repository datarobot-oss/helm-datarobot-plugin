package cmd

import (
	"fmt"
	"strings"

	"github.com/datarobot-oss/helm-datarobot-plugin/pkg/adminchart"
	"github.com/datarobot-oss/helm-datarobot-plugin/pkg/manifest"
	"github.com/datarobot-oss/helm-datarobot-plugin/pkg/render_helper"
	"github.com/spf13/cobra"
)

// defaultClusterReadKindsCopy is a copy of adminchart.DefaultClusterReadKinds used
// as the flag default so the flag value is independent from the package variable.
var defaultClusterReadKindsCopy = append([]string(nil), adminchart.DefaultClusterReadKinds...)

var adminChartCmd = &cobra.Command{
	Use:          "admin-chart",
	Short:        "Generate a helm chart containing all cluster-scoped resources",
	SilenceUsage: true,
	Long: strings.Replace(`
Extract all cluster-scoped resources (CRDs, ClusterRoles, ClusterRoleBindings,
Webhooks, etc.) from a DataRobot Helm chart and package them as a standalone
installable Helm chart.

Use --extra-admin-kinds to force additional kinds (e.g. Role, RoleBinding,
ServiceAccount) into the admin chart for PNC-style restricted RBAC environments
where cluster-admin install handles those privileged resources.

Namespace pinning: when --pipeline-sa is set, the generated pipeline RBAC
(ServiceAccount, RoleBindings) is namespaced to the --namespace value supplied
at GENERATION time. The generated admin chart must be installed into that exact
namespace, and that namespace must equal the namespace used for the future
application chart install. Changing --namespace after generation requires
regenerating the admin chart.

Example:
'''sh
$ helm datarobot admin-chart ./datarobot-prime-11.10.88.tgz \
    --namespace dr \
    --values my-values.yaml \
    --keep-crds=true \
    --extra-admin-kinds Role,RoleBinding,ServiceAccount
'''`, "'", "`", -1),
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		chartPath := args[0]

		namespace := acInput.Namespace
		if namespace == "" {
			namespace = "default"
		}

		renderOpts := &render_helper.RenderOptions{
			Namespace:    namespace,
			ReleaseName:  acInput.ReleaseName,
			KubeVersion:  acInput.KubeVersion,
			IncludeCRDs:  true,
			APIVersions:  acInput.APIVersions,
			IncludeHooks: true,
		}

		if acInput.OpenShift {
			renderOpts.APIVersions = append(renderOpts.APIVersions,
				"security.openshift.io/v1",
				"route.openshift.io/v1",
			)
		}

		var setValues []string
		if f := cmd.Flags().Lookup("set"); f != nil && f.Changed {
			var ferr error
			setValues, ferr = cmd.Flags().GetStringArray("set")
			if ferr != nil {
				setValues = nil
			}
		}

		rendered, err := render_helper.RenderChart(chartPath, acInput.ValueFiles, setValues, renderOpts)
		if err != nil {
			return fmt.Errorf("failed to render chart: %w", err)
		}

		resources, err := manifest.ParseManifests(rendered)
		if err != nil {
			return fmt.Errorf("failed to parse manifests: %w", err)
		}

		result := manifest.Classify(resources, acInput.ExtraAdminKinds)

		for _, w := range result.Warnings {
			cmd.PrintErrln("warning: " + w)
		}

		if len(result.Admin) == 0 {
			return fmt.Errorf("no cluster-scoped resources found in rendered output")
		}

		sourceMeta, err := adminchart.ReadSourceChartMeta(chartPath)
		if err != nil {
			return fmt.Errorf("failed to read source chart metadata: %w", err)
		}

		outputPath := acInput.Output
		if outputPath == "" {
			outputPath = fmt.Sprintf("./datarobot-admin-%s.tgz", sourceMeta.Version)
		}

		chartOpts := adminchart.ChartOptions{
			Name:       "datarobot-admin",
			Version:    sourceMeta.Version,
			AppVersion: sourceMeta.AppVersion,
			SourceName: sourceMeta.Name,
			KeepCRDs:   acInput.KeepCRDs,
		}

		pipelineSA := acInput.PipelineSA
		if pipelineSA != "" {
			// Warn if the admin partition already contains a ServiceAccount with the same
			// name — installing both would fail with a conflict.
			for _, r := range result.Admin {
				if r.Kind == "ServiceAccount" && r.Name == pipelineSA {
					cmd.PrintErrln("warning: --pipeline-sa name " + pipelineSA + " collides with an existing ServiceAccount in the admin partition (added via --extra-admin-kinds); duplicate SA will fail install — choose a different --pipeline-sa name or drop ServiceAccount from --extra-admin-kinds")
					break
				}
			}

			unionRules, err := manifest.RoleRules(result.App)
			if err != nil {
				return fmt.Errorf("failed to compute role-union rules: %w", err)
			}

			clusterReadRules, err := adminchart.ClusterReadRules(acInput.ClusterReadKinds)
			if err != nil {
				return fmt.Errorf("invalid --cluster-read-kinds: %w", err)
			}

			rbac, err := adminchart.BuildPipelineRBAC(adminchart.PipelineRBACOptions{
				SAName:           pipelineSA,
				Namespace:        namespace,
				ReleaseName:      acInput.ReleaseName,
				ClusterReadRules: clusterReadRules,
				UnionRules:       unionRules,
			})
			if err != nil {
				return fmt.Errorf("failed to build pipeline RBAC: %w", err)
			}
			chartOpts.PipelineRBAC = rbac

			var unionSummary string
			if len(unionRules) == 0 {
				unionSummary = "role-union skipped (no Roles in app partition)"
			} else {
				unionSummary = fmt.Sprintf("role-union from %d Role rules", len(unionRules))
			}
			cmd.Printf("Pipeline RBAC: SA %s/%s, cluster-read (%d kinds), %s\n",
				namespace, pipelineSA, len(acInput.ClusterReadKinds), unionSummary)
		}

		adminChart, err := adminchart.BuildChart(result.Admin, chartOpts)
		if err != nil {
			return fmt.Errorf("failed to build admin chart: %w", err)
		}

		if err := adminchart.PackageChart(adminChart, outputPath); err != nil {
			return fmt.Errorf("failed to package chart: %w", err)
		}

		summary := manifest.Summary(result.Admin)
		cmd.Printf("Extracted %d cluster-scoped resources (skipped %d namespaced): %s\n", len(result.Admin), len(result.App), summary)
		cmd.Printf("Admin chart written to: %s\n", outputPath)

		if acInput.Debug {
			cmd.Println("\nResources:")
			for _, r := range result.Admin {
				cmd.Printf("  %s/%s\n", r.Kind, r.Name)
			}
		}

		return nil
	},
}

type adminChartInput struct {
	Namespace        string
	ReleaseName      string
	KubeVersion      string
	Output           string
	ValueFiles       []string
	Values           []string
	APIVersions      []string
	OpenShift        bool
	Debug            bool
	ExtraAdminKinds  []string
	KeepCRDs         bool
	PipelineSA       string
	ClusterReadKinds []string
}

var acInput adminChartInput

func init() {
	rootCmd.AddCommand(adminChartCmd)
	adminChartCmd.Flags().StringVar(&acInput.Namespace, "namespace", "", `Kubernetes namespace for .Release.Namespace template rendering (default: "default"). When --pipeline-sa is set, this value is also used as the namespace for the generated SA and RoleBindings; it must equal the namespace where both the admin chart and the future app chart will be installed.`)
	adminChartCmd.Flags().StringVar(&acInput.ReleaseName, "release-name", "datarobot", "Helm release name")
	adminChartCmd.Flags().StringVar(&acInput.KubeVersion, "kube-version", "v1.32.0", "Kubernetes version for template rendering")
	adminChartCmd.Flags().StringVarP(&acInput.Output, "output", "o", "", "Output path for the admin chart .tgz (default: ./datarobot-admin-<version>.tgz)")
	adminChartCmd.Flags().StringSliceVarP(&acInput.ValueFiles, "values", "f", []string{}, "Specify values in a YAML file (can specify multiple)")
	adminChartCmd.Flags().StringArrayVar(&acInput.Values, "set", []string{}, "Set values on the command line (can specify multiple)")
	adminChartCmd.Flags().StringSliceVar(&acInput.APIVersions, "api-versions", []string{}, "Additional API versions for template rendering")
	adminChartCmd.Flags().BoolVar(&acInput.OpenShift, "openshift", false, "Include OpenShift API versions (security.openshift.io/v1, route.openshift.io/v1)")
	adminChartCmd.Flags().BoolVarP(&acInput.Debug, "debug", "d", false, "Print detailed resource listing")
	adminChartCmd.Flags().StringSliceVar(&acInput.ExtraAdminKinds, "extra-admin-kinds", []string{}, "Resource kinds to force into the admin chart even if namespaced (e.g. Role,RoleBinding,ServiceAccount). Useful for PNC-style restricted RBAC where cluster-admin cannot be assumed.")
	adminChartCmd.Flags().BoolVar(&acInput.KeepCRDs, "keep-crds", true, "Add helm.sh/resource-policy: keep annotation to CRDs in the generated chart to prevent CR data loss on uninstall.")
	adminChartCmd.Flags().StringVar(&acInput.PipelineSA, "pipeline-sa", "", "Name of the limited-privilege ServiceAccount to bootstrap for pipeline use. When set, generates: SA + RoleBinding to built-in admin ClusterRole + cluster-read ClusterRole/CRB (for chart render-time lookup calls) + role-union ClusterRole/RoleBinding (RBAC escalation prevention fix). Empty = feature off. Note: the SA and RoleBindings are namespaced to the --namespace value at GENERATION time; the admin chart must be installed into that namespace and it must equal the future app-install namespace. Note: role-union covers rules from kind: Role only; RoleBindings referencing ClusterRoles are not covered yet.")
	adminChartCmd.Flags().StringSliceVar(&acInput.ClusterReadKinds, "cluster-read-kinds", defaultClusterReadKindsCopy, `"resource.group" specs for the cluster-read ClusterRole rules (e.g. storageclasses.storage.k8s.io,namespaces).`)
}
