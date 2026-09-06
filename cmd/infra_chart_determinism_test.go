package cmd

import (
	"path/filepath"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
)

// Publish-readiness: identical inputs -> content-deterministic chart (spec §10).
//
// This proves generation is CONTENT-deterministic: running `infra-chart`
// twice, in two independent command invocations, over identical inputs
// yields charts with identical metadata (name/version/appVersion) and an
// identical set of templates (same names, same bytes).
//
// Note: this deliberately does NOT compare raw tarball bytes/sha256.
// chartutil.Save stamps wall-clock time.Now() (truncated to whole seconds)
// into the tar header mtimes it writes, so two generate calls that straddle
// a wall-clock second boundary produce different raw tarball bytes even
// though the chart content is identical. That's a Helm-library artifact, not
// a determinism bug in this tool. Raw-tarball byte-reproducibility (needed
// for a signable/publishable output) is tracked as a follow-up, spec §11
// (deferred).
func TestInfraChartDeterministic(t *testing.T) {
	tmp := t.TempDir()
	outA := filepath.Join(tmp, "a.tgz")
	outB := filepath.Join(tmp, "b.tgz")

	_, err := executeCommand(rootCmd, "infra-chart ../tests/charts/crd-test-chart -o "+outA+" --pipeline-sa pipeline --namespace dr")
	assert.NoError(t, err)
	_, err = executeCommand(rootCmd, "infra-chart ../tests/charts/crd-test-chart -o "+outB+" --pipeline-sa pipeline --namespace dr")
	assert.NoError(t, err)

	chartA, err := loader.Load(outA)
	assert.NoError(t, err)
	chartB, err := loader.Load(outB)
	assert.NoError(t, err)

	assert.Equal(t, chartA.Metadata.Name, chartB.Metadata.Name, "chart name must be deterministic")
	assert.Equal(t, chartA.Metadata.Version, chartB.Metadata.Version, "chart version must be deterministic")
	assert.Equal(t, chartA.Metadata.AppVersion, chartB.Metadata.AppVersion, "chart appVersion must be deterministic")

	assert.Equal(t, len(chartA.Templates), len(chartB.Templates), "template count must be deterministic")
	assert.Equal(t, sortedTemplateData(chartA.Templates), sortedTemplateData(chartB.Templates), "template contents must be byte-identical across repeated generation")
}

// sortedTemplateData collapses a chart's templates into a name->data map so
// the comparison is order-independent while still asserting on real byte
// content (a map with duplicate names would silently drop entries, but Helm
// chart templates are unique by Name within a chart).
func sortedTemplateData(files []*chart.File) map[string][]byte {
	sorted := make([]*chart.File, len(files))
	copy(sorted, files)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Name < sorted[j].Name })

	out := make(map[string][]byte, len(sorted))
	for _, f := range sorted {
		out[f.Name] = f.Data
	}
	return out
}
