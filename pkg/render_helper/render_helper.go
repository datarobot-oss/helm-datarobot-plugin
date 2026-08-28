package render_helper

import (
	"fmt"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/cli/values"
	"helm.sh/helm/v3/pkg/getter"
)

type RenderOptions struct {
	Namespace   string
	ReleaseName string
	KubeVersion string
	IncludeCRDs bool
	APIVersions []string
}

// RenderChart preserves the original behavior as a thin wrapper.
func RenderChart(chartPath string, valueFiles []string, Values []string) (string, error) {
	return RenderChartWithOptions(chartPath, valueFiles, Values, &RenderOptions{
		Namespace:   "test",
		ReleaseName: "test-release",
		KubeVersion: "v1.27.0",
		IncludeCRDs: false,
	})
}

func RenderChartWithOptions(chartPath string, valueFiles, Values []string, opts *RenderOptions) (string, error) {
	if opts == nil {
		opts = &RenderOptions{}
	}
	client := action.NewInstall(&action.Configuration{})
	client.ClientOnly = true
	client.DryRun = true
	client.DisableHooks = true
	client.ReleaseName = opts.ReleaseName
	client.Namespace = opts.Namespace
	client.IncludeCRDs = opts.IncludeCRDs
	if len(opts.APIVersions) > 0 {
		client.APIVersions = chartutil.VersionSet(opts.APIVersions)
	}

	kubeVersion := opts.KubeVersion
	if kubeVersion == "" {
		kubeVersion = "v1.27.0"
	}
	parsedKubeVersion, err := chartutil.ParseKubeVersion(kubeVersion)
	if err != nil {
		return "", fmt.Errorf("invalid kube version: %s", err)
	}
	client.KubeVersion = parsedKubeVersion

	valueOpts := &values.Options{
		ValueFiles: valueFiles,
		Values:     Values,
	}

	loadedChart, err := loader.Load(chartPath)
	if err != nil {
		return "", fmt.Errorf("Error loading chart %s: %v", chartPath, err)
	}

	var settings = cli.New()
	p := getter.All(settings)
	vals, err := valueOpts.MergeValues(p)
	if err != nil {
		return "", err
	}

	rel, err := client.Run(loadedChart, vals)
	if err != nil {
		return "", fmt.Errorf("could not render helm chart correctly: %w", err)
	}

	return rel.Manifest, nil
}
