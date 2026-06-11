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
	Name         string
	Version      string
	AppVersion   string
	SourceName   string              // name of the source chart (e.g. "datarobot-prime") — used in Description
	KeepCRDs     bool                // when true, add annotation helm.sh/resource-policy: keep to every CustomResourceDefinition
	PipelineRBAC []manifest.Resource // when non-empty, emitted as templates/pipeline-rbac.yaml (single auditable unit)
}

// BuildChart creates a *chart.Chart from a slice of resources.
// Resources are grouped by kind; each kind gets one template file.
func BuildChart(resources []manifest.Resource, opts ChartOptions) (*chart.Chart, error) {
	if len(resources) == 0 {
		return nil, fmt.Errorf("no resources provided")
	}

	// Process each resource: strip hook annotations, apply keep policy for CRDs.
	processed := make([]manifest.Resource, 0, len(resources))
	for _, r := range resources {
		stripped, err := r.StripHelmHookAnnotations()
		if err != nil {
			return nil, fmt.Errorf("strip hook annotations for %s/%s: %w", r.Kind, r.Name, err)
		}
		if opts.KeepCRDs && stripped.Kind == "CustomResourceDefinition" {
			kept, err := stripped.WithAnnotation("helm.sh/resource-policy", "keep")
			if err != nil {
				return nil, fmt.Errorf("add keep annotation to %s: %w", stripped.Name, err)
			}
			stripped = kept
		}
		processed = append(processed, stripped)
	}

	grouped := manifest.GroupByKind(processed)

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
		// Escape Go-template braces so Helm does not try to evaluate them at install time.
		content = strings.ReplaceAll(content, "{{", "{{`{{`}}")
		name := fmt.Sprintf("templates/%ss.yaml", strings.ToLower(kind))
		templates = append(templates, &chart.File{
			Name: name,
			Data: []byte(content),
		})
	}

	// Emit pipeline RBAC as one auditable template file (not grouped by kind,
	// not processed through StripHelmHookAnnotations/KeepCRDs — generated clean).
	if len(opts.PipelineRBAC) > 0 {
		rbacParts := make([]string, 0, len(opts.PipelineRBAC))
		for _, r := range opts.PipelineRBAC {
			rbacParts = append(rbacParts, r.RawYAML)
		}
		rbacContent := strings.Join(rbacParts, "\n---\n")
		rbacContent = strings.ReplaceAll(rbacContent, "{{", "{{`{{`}}")
		templates = append(templates, &chart.File{
			Name: "templates/pipeline-rbac.yaml",
			Data: []byte(rbacContent),
		})
	}

	sourceName := opts.SourceName
	if sourceName == "" {
		sourceName = opts.Name
	}

	c := &chart.Chart{
		Metadata: &chart.Metadata{
			APIVersion:  "v2",
			Name:        opts.Name,
			Version:     opts.Version,
			AppVersion:  opts.AppVersion,
			Type:        "application",
			Description: fmt.Sprintf("Cluster-scoped admin resources extracted from %s %s", sourceName, opts.Version),
		},
		Templates: templates,
	}

	return c, nil
}

// PackageChart saves c as a .tgz at outputPath.
func PackageChart(c *chart.Chart, outputPath string) error {
	outDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return fmt.Errorf("create output dir: %w", err)
	}

	// Save directly into the output directory — same filesystem, no cross-device rename.
	saved, err := chartutil.Save(c, outDir)
	if err != nil {
		return fmt.Errorf("save chart: %w", err)
	}

	if filepath.Clean(saved) != filepath.Clean(outputPath) {
		if err := os.Rename(saved, outputPath); err != nil {
			return fmt.Errorf("move chart to output: %w", err)
		}
	}

	return nil
}
