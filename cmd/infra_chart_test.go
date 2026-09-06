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

func TestCommandInfraChart(t *testing.T) {
	tmp := t.TempDir()
	out := filepath.Join(tmp, "infra.tgz")
	_, err := executeCommand(rootCmd, "infra-chart ../tests/charts/crd-test-chart -o "+out)
	assert.NoError(t, err)

	c, err := loader.Load(out)
	assert.NoError(t, err)
	assert.Equal(t, "datarobot-infra", c.Metadata.Name)
	assert.Equal(t, "9.9.9", c.Metadata.Version)

	rendered, err := render_helper.RenderChart(out, []string{}, []string{}, nil)
	assert.NoError(t, err)
	assert.Equal(t, 2, strings.Count(rendered, "kind: CustomResourceDefinition"))
	assert.NotContains(t, rendered, "kind: Deployment")
	assert.Contains(t, rendered, "helm.sh/resource-policy: keep")
	assert.Contains(t, rendered, "{{ .foo }}")
}

func TestCommandInfraChartNoKeep(t *testing.T) {
	tmp := t.TempDir()
	out := filepath.Join(tmp, "infra.tgz")
	_, err := executeCommand(rootCmd, "infra-chart ../tests/charts/crd-test-chart -o "+out+" --keep-crds=false")
	assert.NoError(t, err)
	rendered, err := render_helper.RenderChart(out, []string{}, []string{}, nil)
	assert.NoError(t, err)
	assert.NotContains(t, rendered, "helm.sh/resource-policy: keep")
}

func TestCommandInfraChartDefaultOutputName(t *testing.T) {
	cwd, _ := os.Getwd()
	defer os.Remove(filepath.Join(cwd, "datarobot-infra-9.9.9.tgz"))
	_, err := executeCommand(rootCmd, "infra-chart ../tests/charts/crd-test-chart")
	assert.NoError(t, err)
	_, statErr := os.Stat(filepath.Join(cwd, "datarobot-infra-9.9.9.tgz"))
	assert.NoError(t, statErr)
}

func TestCommandInfraChartNoResourcesError(t *testing.T) {
	tmp := t.TempDir()
	out := filepath.Join(tmp, "infra.tgz")
	_, err := executeCommand(rootCmd, "infra-chart ../tests/charts/test-chart6 -o "+out)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no cluster-scoped resources")
	_, statErr := os.Stat(out)
	assert.Error(t, statErr)
}
