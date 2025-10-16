package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/spf13/cobra"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/release"
)

var upgradeCmd = &cobra.Command{
	Use:          "upgrade [CHART]",
	Short:        "validate chart upgrade compatibility",
	SilenceUsage: true,
	Long: strings.Replace(`
This command validates whether a chart upgrade is possible by comparing 
the version of the supplied chart against the currently deployed chart 
in the Kubernetes namespace.

The command checks:
- If the chart is currently installed in the namespace
- If the new version is greater than or equal to the old version

Example:
'''sh
$ helm datarobot upgrade tests/charts/test-chart1/ -n default
Chart test-chart1 can be upgraded from version 0.1.0 to 0.2.0

$ helm datarobot upgrade ./my-chart -n production
Error: chart my-chart is not installed in namespace production
'''`, "'", "`", -1),
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		chartPath := args[0]

		// Get namespace from flag
		namespace, err := cmd.Flags().GetString("namespace")
		if err != nil {
			return fmt.Errorf("error getting namespace flag: %w", err)
		}

		// Default to "default" namespace if not specified
		if namespace == "" {
			namespace = "default"
		}

		return runUpgradeValidation(cmd, chartPath, namespace)
	},
}

func runUpgradeValidation(cmd *cobra.Command, chartPath, namespace string) error {
	// Validate that the chart path is a local directory or file
	if strings.HasPrefix(chartPath, "oci://") || strings.HasPrefix(chartPath, "http://") || strings.HasPrefix(chartPath, "https://") {
		return fmt.Errorf("only local chart paths are supported, got: %s", chartPath)
	}

	// Load the chart
	loadedChart, err := loader.Load(chartPath)
	if err != nil {
		return fmt.Errorf("error loading chart %s: %w", chartPath, err)
	}

	chartName := loadedChart.Metadata.Name
	newVersionStr := loadedChart.Metadata.Version

	// Parse the new version
	newVersion, err := semver.NewVersion(newVersionStr)
	if err != nil {
		return fmt.Errorf("invalid chart version %s: %w", newVersionStr, err)
	}

	// Setup Helm action configuration
	settings := cli.New()
	settings.SetNamespace(namespace)

	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(settings.RESTClientGetter(), namespace, os.Getenv("HELM_DRIVER"), func(format string, v ...interface{}) {
		// Silent debug function
	}); err != nil {
		return fmt.Errorf("error initializing helm action config: %w", err)
	}

	// List releases in the namespace
	listClient := action.NewList(actionConfig)
	listClient.All = true // Include all releases, even failed ones

	releases, err := listClient.Run()
	if err != nil {
		return fmt.Errorf("error listing releases in namespace %s: %w", namespace, err)
	}

	// Find the release with matching chart name
	var currentRelease *release.Release
	for _, rel := range releases {
		if rel.Chart != nil && rel.Chart.Metadata != nil {
			if rel.Chart.Metadata.Name == chartName {
				currentRelease = rel
				break
			}
		}
	}

	// Check if chart is installed
	if currentRelease == nil {
		return fmt.Errorf("chart %s is not installed in namespace %s", chartName, namespace)
	}

	oldVersionStr := currentRelease.Chart.Metadata.Version

	// Parse the old version
	oldVersion, err := semver.NewVersion(oldVersionStr)
	if err != nil {
		return fmt.Errorf("invalid installed chart version %s: %w", oldVersionStr, err)
	}

	// Compare versions
	if newVersion.LessThan(oldVersion) {
		return fmt.Errorf("cannot downgrade chart %s from version %s to %s", chartName, oldVersionStr, newVersionStr)
	}

	// Success case
	if newVersion.Equal(oldVersion) {
		cmd.Printf("Chart %s is already at version %s in namespace %s\n", chartName, newVersionStr, namespace)
	} else {
		cmd.Printf("Chart %s can be upgraded from version %s to %s in namespace %s\n", chartName, oldVersionStr, newVersionStr, namespace)
	}

	return nil
}

func init() {
	rootCmd.AddCommand(upgradeCmd)

	// Add namespace flag that matches helm's behavior
	upgradeCmd.Flags().StringP("namespace", "n", "", "namespace scope for this request")

	// Setup upgrade action to bind all standard helm upgrade flags
	settings := cli.New()
	actionConfig := new(action.Configuration)
	upgradeAction := action.NewUpgrade(actionConfig)

	// Bind all upgrade flags to the command
	// This dynamically inherits all flags from the helm upgrade command
	addUpgradeFlags(upgradeCmd, upgradeAction, settings)
}

// addUpgradeFlags adds all standard helm upgrade flags to the command
func addUpgradeFlags(cmd *cobra.Command, client *action.Upgrade, settings *cli.EnvSettings) {
	f := cmd.Flags()

	// Core upgrade flags that exist on action.Upgrade
	f.BoolVar(&client.Install, "install", false, "if a release by this name doesn't already exist, run an install")
	f.BoolVar(&client.Devel, "devel", false, "use development versions, too. Equivalent to version '>0.0.0-0'. If --version is set, this is ignored")
	f.BoolVar(&client.DryRun, "dry-run", false, "simulate an upgrade")
	f.BoolVar(&client.Recreate, "recreate-pods", false, "performs pods restart for the resource if applicable")
	f.BoolVar(&client.Force, "force", false, "force resource updates through a replacement strategy")
	f.BoolVar(&client.DisableHooks, "no-hooks", false, "disable pre/post upgrade hooks")
	f.BoolVar(&client.DisableOpenAPIValidation, "disable-openapi-validation", false, "if set, the upgrade process will not validate rendered templates against the Kubernetes OpenAPI Schema")
	f.BoolVar(&client.SkipCRDs, "skip-crds", false, "if set, no CRDs will be installed when an upgrade is performed with install flag enabled. By default, CRDs are installed if not already present, when an upgrade is performed with install flag enabled")
	f.DurationVar(&client.Timeout, "timeout", 300000000000, "time to wait for any individual Kubernetes operation (like Jobs for hooks)")
	f.BoolVar(&client.Wait, "wait", false, "if set, will wait until all Pods, PVCs, Services, and minimum number of Pods of a Deployment, StatefulSet, or ReplicaSet are in a ready state before marking the release as successful. It will wait for as long as --timeout")
	f.BoolVar(&client.WaitForJobs, "wait-for-jobs", false, "if set and --wait enabled, will wait until all Jobs have been completed before marking the release as successful. It will wait for as long as --timeout")
	f.BoolVar(&client.Atomic, "atomic", false, "if set, upgrade process rolls back changes made in case of failed upgrade. The --wait flag will be set automatically if --atomic is used")
	f.IntVar(&client.MaxHistory, "max-history", 10, "limit the maximum number of revisions saved per release. Use 0 for no limit")
	f.BoolVar(&client.CleanupOnFail, "cleanup-on-fail", false, "allow deletion of new resources created in this upgrade when upgrade fails")
	f.BoolVar(&client.SubNotes, "render-subchart-notes", false, "if set, render subchart notes along with the parent")
	f.StringVar(&client.Description, "description", "", "add a custom description")
	f.BoolVar(&client.ReuseValues, "reuse-values", false, "when upgrading, reuse the last release's values and merge in any overrides from the command line via --set and -f. If '--reset-values' is specified, this is ignored")
	f.BoolVar(&client.ResetValues, "reset-values", false, "when upgrading, reset the values to the ones built into the chart")
	f.BoolVar(&client.ResetThenReuseValues, "reset-then-reuse-values", false, "when upgrading, reset the values to the ones built into the chart, apply the last release's values and merge in any overrides from the command line via --set and -f. If '--reset-values' or '--reuse-values' is specified, this is ignored")
	f.StringVar(&client.Version, "version", "", "specify a version constraint for the chart version to use. This constraint can be a specific tag (e.g. 1.1.1) or it may reference a valid range (e.g. ^2.0.0). If this is not specified, the latest version is used")

	var createNamespace bool
	f.BoolVar(&createNamespace, "create-namespace", false, "if --install is set, create the release namespace if not present")
	f.BoolVar(&client.EnableDNS, "enable-dns", false, "enable DNS lookups when rendering templates")
	f.BoolVar(&client.DependencyUpdate, "dependency-update", false, "update dependencies if they are missing before installing the chart")

	// TLS/Auth flags
	f.StringVar(&client.CertFile, "cert-file", "", "identify HTTPS client using this SSL certificate file")
	f.StringVar(&client.KeyFile, "key-file", "", "identify HTTPS client using this SSL key file")
	f.StringVar(&client.CaFile, "ca-file", "", "verify certificates of HTTPS-enabled servers using this CA bundle")
	f.BoolVar(&client.InsecureSkipTLSverify, "insecure-skip-tls-verify", false, "skip tls certificate checks for the chart download")
	f.BoolVar(&client.PlainHTTP, "plain-http", false, "use insecure HTTP connections for the chart download")
	f.StringVar(&client.Username, "username", "", "chart repository username where to locate the requested chart")
	f.StringVar(&client.Password, "password", "", "chart repository password where to locate the requested chart")
	f.BoolVar(&client.PassCredentialsAll, "pass-credentials", false, "pass credentials to all domains")

	// Post renderer - special handling as it's not a simple string
	var postRenderer string
	f.StringVar(&postRenderer, "post-renderer", "", "the path to an executable to be used for post rendering. If it exists in $PATH, the binary will be used, otherwise it will try to look for the executable at the given path")

	// Value flags - use standard string arrays for now
	var valueFiles []string
	var setValues []string
	var setStringValues []string
	var setFileValues []string
	var setJSONValues []string
	var setLiteralValues []string

	f.StringArrayVarP(&valueFiles, "values", "f", []string{}, "specify values in a YAML file or a URL (can specify multiple)")
	f.StringArrayVar(&setValues, "set", []string{}, "set values on the command line (can specify multiple or separate values with commas: key1=val1,key2=val2)")
	f.StringArrayVar(&setStringValues, "set-string", []string{}, "set STRING values on the command line (can specify multiple or separate values with commas: key1=val1,key2=val2)")
	f.StringArrayVar(&setFileValues, "set-file", []string{}, "set values from respective files specified via the command line (can specify multiple or separate values with commas: key1=path1,key2=path2)")
	f.StringArrayVar(&setJSONValues, "set-json", []string{}, "set JSON values on the command line (can specify multiple or separate values with commas: key1=jsonval1,key2=jsonval2)")
	f.StringArrayVar(&setLiteralValues, "set-literal", []string{}, "set a literal STRING value on the command line")

	// Labels - map format
	var labels map[string]string
	f.StringToStringVarP(&labels, "labels", "l", nil, "Labels that would be added to release metadata")
}
