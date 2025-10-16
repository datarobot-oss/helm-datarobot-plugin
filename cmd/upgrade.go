package cmd

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
)

// UpgradeAnnotation represents a structured upgrade annotation
type UpgradeAnnotation struct {
	Source      string `yaml:"source"`
	Target      string `yaml:"target"`
	Action      string `yaml:"action"`
	Condition   string `yaml:"condition,omitempty"`
	Description string `yaml:"description"`
}

var upgradeCmd = &cobra.Command{
	Use:          "upgrade [RELEASE] [CHART]",
	Short:        "validate chart upgrade compatibility",
	SilenceUsage: true,
	Long: strings.Replace(`
This command validates whether a chart upgrade is possible by comparing 
the version of the supplied chart against the currently deployed release 
in the Kubernetes namespace.

The command checks:
- If the release is currently installed in the namespace
- If the new version is greater than or equal to the old version

Example:
'''sh
$ helm datarobot upgrade dr tests/charts/test-chart1/ -n default
Release dr can be upgraded from version 0.1.0 to 0.2.0

# Database schema migration required
./scripts/migrate-db.sh

# Run backup before upgrade
./scripts/backup.sh
./scripts/verify-backup.sh

$ helm datarobot upgrade dr ./my-chart -n production
Error: release dr is not installed in namespace production
'''`, "'", "`", -1),
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		releaseName := args[0]
		chartPath := args[1]

		// Get namespace from flag
		namespace, err := cmd.Flags().GetString("namespace")
		if err != nil {
			return fmt.Errorf("error getting namespace flag: %w", err)
		}

		// Default to "default" namespace if not specified
		if namespace == "" {
			namespace = "default"
		}

		return runUpgradeValidation(cmd, releaseName, chartPath, namespace)
	},
}

func runUpgradeValidation(cmd *cobra.Command, releaseName, chartPath, namespace string) error {
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

	// Get the specific release
	getClient := action.NewGet(actionConfig)
	currentRelease, err := getClient.Run(releaseName)
	if err != nil {
		return fmt.Errorf("release %s is not installed in namespace %s: %w", releaseName, namespace, err)
	}

	// Verify the release is using the same chart
	if currentRelease.Chart == nil || currentRelease.Chart.Metadata == nil {
		return fmt.Errorf("release %s has invalid chart metadata", releaseName)
	}

	installedChartName := currentRelease.Chart.Metadata.Name
	if installedChartName != chartName {
		return fmt.Errorf("release %s is using chart %s, but trying to upgrade with chart %s", releaseName, installedChartName, chartName)
	}

	oldVersionStr := currentRelease.Chart.Metadata.Version

	// Parse the old version
	oldVersion, err := semver.NewVersion(oldVersionStr)
	if err != nil {
		return fmt.Errorf("invalid installed chart version %s: %w", oldVersionStr, err)
	}

	// Compare versions
	if newVersion.LessThan(oldVersion) {
		return fmt.Errorf("cannot downgrade release %s from version %s to %s", releaseName, oldVersionStr, newVersionStr)
	}

	// Success case
	if newVersion.Equal(oldVersion) {
		cmd.Printf("Release %s is already at version %s in namespace %s\n", releaseName, newVersionStr, namespace)
	} else {
		cmd.Printf("Release %s can be upgraded from version %s to %s in namespace %s\n", releaseName, oldVersionStr, newVersionStr, namespace)

		// Display upgrade annotations if any exist
		upgradeAnnotations := extractAndParseUpgradeAnnotations(loadedChart)
		if len(upgradeAnnotations) > 0 {
			// Sort keys for consistent output order
			var keys []string
			for key := range upgradeAnnotations {
				keys = append(keys, key)
			}
			sort.Strings(keys)

			// Filter and display matching annotations
			hasDisplayed := false
			for _, key := range keys {
				annotation := upgradeAnnotations[key]

				// Check if annotation should be displayed based on version matching
				if shouldDisplayAnnotation(annotation, oldVersionStr, newVersionStr) {
					if !hasDisplayed {
						cmd.Printf("\n")
						hasDisplayed = true
					}
					cmd.Printf("# %s\n", annotation.Description)
					cmd.Printf("%s\n\n", annotation.Action)
				}
			}
		}
	}

	return nil
}

// extractAndParseUpgradeAnnotations extracts and parses all annotations from the chart that have the "upgrade.datarobot.com/" prefix
func extractAndParseUpgradeAnnotations(chart *chart.Chart) map[string]UpgradeAnnotation {
	upgradeAnnotations := make(map[string]UpgradeAnnotation)

	if chart.Metadata == nil || chart.Metadata.Annotations == nil {
		return upgradeAnnotations
	}

	for key, value := range chart.Metadata.Annotations {
		if strings.HasPrefix(key, "upgrade.datarobot.com/") {
			var annotation UpgradeAnnotation
			err := yaml.Unmarshal([]byte(value), &annotation)
			if err != nil {
				// Skip invalid YAML annotations with warning
				fmt.Fprintf(os.Stderr, "Warning: failed to parse annotation %s: %v\n", key, err)
				continue
			}

			// Validate required fields
			if annotation.Action == "" || annotation.Description == "" {
				fmt.Fprintf(os.Stderr, "Warning: annotation %s missing required fields (action or description)\n", key)
				continue
			}

			upgradeAnnotations[key] = annotation
		}
	}

	return upgradeAnnotations
}

// matchesVersionConstraint checks if a version matches a constraint (exact, range, or wildcard)
func matchesVersionConstraint(version, constraint string) (bool, error) {
	// Empty constraint matches any version
	if constraint == "" {
		return true, nil
	}

	// Try wildcard matching first (contains x or *)
	if strings.Contains(constraint, "x") || strings.Contains(constraint, "*") {
		return matchesWildcardVersion(version, constraint), nil
	}

	// Try as semver constraint (range like ">=1.0.0 <2.0.0")
	if strings.ContainsAny(constraint, "><~^") {
		return matchesSemverConstraint(version, constraint)
	}

	// Exact version match
	v1, err := semver.NewVersion(version)
	if err != nil {
		return false, err
	}
	v2, err := semver.NewVersion(constraint)
	if err != nil {
		return false, err
	}
	return v1.Equal(v2), nil
}

// matchesWildcardVersion checks if version matches wildcard pattern
func matchesWildcardVersion(version, pattern string) bool {
	if pattern == "*" {
		return true
	}

	// Replace x with * for consistency
	pattern = strings.ReplaceAll(pattern, "x", "*")

	// Build regex: escape dots, replace * with \d+(\.\d+)*
	escapedPattern := regexp.QuoteMeta(pattern)
	escapedPattern = strings.ReplaceAll(escapedPattern, "\\*", "\\d+(\\.\\d+)*")

	regexPattern := "^" + escapedPattern + "$"
	matched, _ := regexp.MatchString(regexPattern, version)
	return matched
}

// matchesSemverConstraint checks if version matches semver constraint
func matchesSemverConstraint(version, constraint string) (bool, error) {
	v, err := semver.NewVersion(version)
	if err != nil {
		return false, err
	}

	c, err := semver.NewConstraint(constraint)
	if err != nil {
		return false, err
	}

	return c.Check(v), nil
}

// shouldDisplayAnnotation determines if annotation should be displayed based on version matching
func shouldDisplayAnnotation(annotation UpgradeAnnotation, installedVersion, newVersion string) bool {
	// Both empty → always display
	if annotation.Source == "" && annotation.Target == "" {
		return true
	}

	// Check source (installed version)
	if annotation.Source != "" {
		matches, err := matchesVersionConstraint(installedVersion, annotation.Source)
		if err != nil || !matches {
			return false
		}
	}

	// Check target (new version)
	if annotation.Target != "" {
		matches, err := matchesVersionConstraint(newVersion, annotation.Target)
		if err != nil || !matches {
			return false
		}
	}

	return true
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
