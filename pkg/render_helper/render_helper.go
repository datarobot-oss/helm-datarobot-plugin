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

func RenderChart(chartPath string, valueFiles []string, Values []string) (string, error) {
	client := action.NewInstall(&action.Configuration{})
	client.ClientOnly = true
	client.DryRun = true
	client.ReleaseName = "test-release"
	client.IncludeCRDs = false
	client.Namespace = "test"
	client.DisableHooks = true
	parsedKubeVersion, err := chartutil.ParseKubeVersion("v1.27.0")
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
	values, err := valueOpts.MergeValues(p)
	if err != nil {
		return "", err
	}

	// Render chart.
	rel, err := client.Run(loadedChart, values)
	if err != nil {
		return "", fmt.Errorf("could not render helm chart correctly: %w", err)
	}

	return rel.Manifest, nil
}
