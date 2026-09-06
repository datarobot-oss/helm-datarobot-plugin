package render_helper

import (
	"fmt"
	"strings"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/cli/values"
	"helm.sh/helm/v3/pkg/getter"
)

// RenderOptions controls how a chart is rendered. If nil is passed to RenderChart, defaultOptions() is used.
type RenderOptions struct {
	Namespace    string
	ReleaseName  string
	KubeVersion  string
	IncludeCRDs  bool
	APIVersions  []string
	IncludeHooks bool // when true, hook manifests are appended to the returned output
}

func defaultOptions() *RenderOptions {
	return &RenderOptions{
		Namespace:    "test",
		ReleaseName:  "test-release",
		KubeVersion:  "v1.27.0",
		IncludeCRDs:  false,
		APIVersions:  nil,
		IncludeHooks: false,
	}
}

func RenderChart(chartPath string, valueFiles []string, Values []string, opts *RenderOptions) (string, error) {
	if opts == nil {
		opts = defaultOptions()
	}

	client := action.NewInstall(&action.Configuration{})
	client.ClientOnly = true
	client.DryRun = true
	client.ReleaseName = opts.ReleaseName
	client.IncludeCRDs = opts.IncludeCRDs
	client.Namespace = opts.Namespace
	client.DisableHooks = !opts.IncludeHooks

	parsedKubeVersion, err := chartutil.ParseKubeVersion(opts.KubeVersion)
	if err != nil {
		return "", fmt.Errorf("invalid kube version: %s", err)
	}
	client.KubeVersion = parsedKubeVersion

	if len(opts.APIVersions) > 0 {
		client.APIVersions = opts.APIVersions
	}

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

	out := rel.Manifest

	if opts.IncludeHooks && rel.Hooks != nil {
		var hookParts []string
		for _, h := range rel.Hooks {
			if strings.TrimSpace(h.Manifest) != "" {
				hookParts = append(hookParts, h.Manifest)
			}
		}
		if len(hookParts) > 0 {
			out = out + "\n---\n" + strings.Join(hookParts, "\n---\n")
		}
	}

	return out, nil
}
