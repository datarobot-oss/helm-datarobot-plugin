package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/datarobot-oss/helm-datarobot-plugin/pkg/render_helper"
	"github.com/stretchr/testify/assert"
	"helm.sh/helm/v3/pkg/chart/loader"
)

func TestCommandCrdChart(t *testing.T) {
	out := filepath.Join(t.TempDir(), "crds.tgz")
	output, err := executeCommand(rootCmd,
		"crd-chart ../tests/charts/crd-test-chart -o "+out)
	assert.NoError(t, err)
	assert.Contains(t, output, "Extracted 2 CRDs")

	// Generated chart loads and contains exactly the two CRDs, no noise.
	c, err := loader.Load(out)
	assert.NoError(t, err)
	assert.Equal(t, "datarobot-infra", c.Metadata.Name)
	assert.Equal(t, "9.9.9", c.Metadata.Version) // copied from source

	// Render the GENERATED chart: it must template cleanly (braces escaped)
	// and yield exactly 2 CRDs and no Deployment/ClusterRole.
	rendered, err := render_helper.RenderChart(out, []string{}, []string{})
	assert.NoError(t, err)
	assert.Equal(t, 2, strings.Count(rendered, "kind: CustomResourceDefinition"))
	assert.NotContains(t, rendered, "kind: Deployment")
	assert.NotContains(t, rendered, "kind: ClusterRole")
	assert.Contains(t, rendered, "helm.sh/resource-policy: keep")
	// The literal brace survived as data, not evaluated as a template.
	assert.Contains(t, rendered, "{{ .foo }}")
}

func TestCommandCrdChartNoKeep(t *testing.T) {
	out := filepath.Join(t.TempDir(), "crds.tgz")
	_, err := executeCommand(rootCmd,
		"crd-chart ../tests/charts/crd-test-chart -o "+out+" --keep-crds=false")
	assert.NoError(t, err)
	rendered, err := render_helper.RenderChart(out, []string{}, []string{})
	assert.NoError(t, err)
	assert.NotContains(t, rendered, "helm.sh/resource-policy: keep")
}

func TestCommandCrdChartDefaultOutputName(t *testing.T) {
	// No -o: default path ./datarobot-infra-<srcVersion>.tgz in cwd.
	defer os.Remove("datarobot-infra-9.9.9.tgz")
	output, err := executeCommand(rootCmd, "crd-chart ../tests/charts/crd-test-chart")
	assert.NoError(t, err)
	assert.Contains(t, output, "Extracted 2 CRDs")
	_, statErr := os.Stat("datarobot-infra-9.9.9.tgz")
	assert.NoError(t, statErr)
}

func TestCommandCrdChartNoCRDsError(t *testing.T) {
	out := filepath.Join(t.TempDir(), "out.tgz")
	_, err := executeCommand(rootCmd,
		"crd-chart ../tests/charts/test-chart6 -o "+out)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no CRDs found")

	// No tarball written on hard-error path.
	_, statErr := os.Stat(out)
	assert.Error(t, statErr)
}
