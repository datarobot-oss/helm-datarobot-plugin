package adminchart

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/datarobot-oss/helm-datarobot-plugin/pkg/manifest"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chartutil"
)

// ChartOptions holds metadata for the generated admin chart.
type ChartOptions struct {
	Name       string
	Version    string
	AppVersion string
}

// BuildChart creates a *chart.Chart from a slice of resources.
// Resources are grouped by kind; each kind gets one template file.
func BuildChart(resources []manifest.Resource, opts ChartOptions) (*chart.Chart, error) {
	if len(resources) == 0 {
		return nil, fmt.Errorf("no resources provided")
	}

	grouped := manifest.GroupByKind(resources)

	kinds := make([]string, 0, len(grouped))
	for k := range grouped {
		kinds = append(kinds, k)
	}
	sort.Strings(kinds)

	var templates []*chart.File
	for _, kind := range kinds {
		parts := make([]string, 0, len(grouped[kind]))
		for _, r := range grouped[kind] {
			parts = append(parts, r.RawYAML)
		}
		content := strings.Join(parts, "\n---\n")
		name := fmt.Sprintf("templates/%ss.yaml", strings.ToLower(kind))
		templates = append(templates, &chart.File{
			Name: name,
			Data: []byte(content),
		})
	}

	c := &chart.Chart{
		Metadata: &chart.Metadata{
			APIVersion:  "v2",
			Name:        opts.Name,
			Version:     opts.Version,
			AppVersion:  opts.AppVersion,
			Type:        "application",
			Description: fmt.Sprintf("Cluster-scoped admin resources for %s %s", opts.Name, opts.Version),
		},
		Templates: templates,
	}

	return c, nil
}

// PackageChart saves c as a .tgz at outputPath.
func PackageChart(c *chart.Chart, outputPath string) error {
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("create output dir: %w", err)
	}

	tmpDir, err := os.MkdirTemp("", "adminchart-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	saved, err := chartutil.Save(c, tmpDir)
	if err != nil {
		return fmt.Errorf("save chart: %w", err)
	}

	if err := os.Rename(saved, outputPath); err != nil {
		return fmt.Errorf("move chart to output: %w", err)
	}

	return nil
}
