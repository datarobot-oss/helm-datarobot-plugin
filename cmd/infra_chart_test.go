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
	// envoy (crds/), notebooks (templates/), widgets (subchart templates/)
	assert.Equal(t, 3, strings.Count(rendered, "kind: CustomResourceDefinition"))
	assert.NotContains(t, rendered, "kind: Deployment")
	assert.Contains(t, rendered, "helm.sh/resource-policy: keep")
	assert.Contains(t, rendered, "{{ .foo }}")
}

// CRD-013: a values file / --set that disables the *global* half of the
// installCRDs gate must not silently drop templated CRDs from the infra chart.
func TestCommandInfraChartForcesGlobalInstallCRDs(t *testing.T) {
	tmp := t.TempDir()
	out := filepath.Join(tmp, "infra.tgz")
	output, err := executeCommand(rootCmd, "infra-chart ../tests/charts/crd-test-chart -o "+out+" --set global.installCRDs=false")
	assert.NoError(t, err)
	rendered, err := render_helper.RenderChart(out, []string{}, []string{}, nil)
	assert.NoError(t, err)
	assert.Equal(t, 3, strings.Count(rendered, "kind: CustomResourceDefinition"),
		"global.installCRDs=false must not drop CRDs from the extracted infra chart")
	assert.Contains(t, output, "global.installCRDs",
		"overriding global.installCRDs=false must be logged, not silent")
}

// CRD-013: a subchart's *local* installCRDs=false must not silently drop its
// CRD — the discovery walk must force each enabled subchart's local key true.
func TestCommandInfraChartForcesSubchartInstallCRDs(t *testing.T) {
	tmp := t.TempDir()
	out := filepath.Join(tmp, "infra.tgz")
	output, err := executeCommand(rootCmd, "infra-chart ../tests/charts/crd-test-chart -o "+out+" --set crd-subchart.installCRDs=false")
	assert.NoError(t, err)
	rendered, err := render_helper.RenderChart(out, []string{}, []string{}, nil)
	assert.NoError(t, err)
	assert.Equal(t, 3, strings.Count(rendered, "kind: CustomResourceDefinition"),
		"a subchart's local installCRDs=false must not drop its CRD from the infra chart")
	assert.Contains(t, rendered, "widgets.datarobot.com")
	assert.Contains(t, output, "crd-subchart.installCRDs",
		"overriding a subchart's installCRDs=false must be logged, not silent")
}

// CRD-013: forcing installCRDs must NOT resurrect a subchart the user disabled
// via enabled=false (topology flag), which is honored differently from the
// install-ownership installCRDs flag.
func TestCommandInfraChartDisabledSubchartExcluded(t *testing.T) {
	tmp := t.TempDir()
	out := filepath.Join(tmp, "infra.tgz")
	_, err := executeCommand(rootCmd, "infra-chart ../tests/charts/crd-test-chart -o "+out+" --set crd-subchart.enabled=false")
	assert.NoError(t, err)
	rendered, err := render_helper.RenderChart(out, []string{}, []string{}, nil)
	assert.NoError(t, err)
	assert.Equal(t, 2, strings.Count(rendered, "kind: CustomResourceDefinition"),
		"forcing installCRDs must not resurrect a disabled subchart's CRD")
	assert.NotContains(t, rendered, "widgets.datarobot.com")
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

func TestCommandInfraChartPipelineSA(t *testing.T) {
	tmp := t.TempDir()
	out := filepath.Join(tmp, "infra.tgz")
	_, err := executeCommand(rootCmd, "infra-chart ../tests/charts/crd-test-chart -o "+out+" --pipeline-sa pipeline --namespace dr --release-name datarobot")
	assert.NoError(t, err)

	rendered, err := render_helper.RenderChart(out, []string{}, []string{}, nil)
	assert.NoError(t, err)
	assert.Contains(t, rendered, "kind: ServiceAccount")
	assert.Contains(t, rendered, "name: pipeline")
	assert.Contains(t, rendered, "datarobot-pipeline-cluster-read")
}

func TestCommandInfraChartCRDAggregationDefaultOn(t *testing.T) {
	tmp := t.TempDir()
	out := filepath.Join(tmp, "infra.tgz")
	_, err := executeCommand(rootCmd, "infra-chart ../tests/charts/crd-test-chart -o "+out)
	assert.NoError(t, err)
	rendered, err := render_helper.RenderChart(out, []string{}, []string{}, nil)
	assert.NoError(t, err)
	// crd-test-chart ships Namespaced CRDs -> aggregate ClusterRole present by default.
	assert.Contains(t, rendered, "datarobot-crd-edit-aggregate")
	assert.Contains(t, rendered, "rbac.authorization.k8s.io/aggregate-to-admin")
}

func TestCommandInfraChartCRDAggregationDisabled(t *testing.T) {
	tmp := t.TempDir()
	out := filepath.Join(tmp, "infra.tgz")
	_, err := executeCommand(rootCmd, "infra-chart ../tests/charts/crd-test-chart -o "+out+" --crd-aggregation=false")
	assert.NoError(t, err)
	rendered, err := render_helper.RenderChart(out, []string{}, []string{}, nil)
	assert.NoError(t, err)
	assert.NotContains(t, rendered, "datarobot-crd-edit-aggregate")
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
