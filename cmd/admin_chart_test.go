package cmd

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"helm.sh/helm/v3/pkg/chart/loader"
)

func TestCommandAdminChart(t *testing.T) {
	tmpDir := t.TempDir()
	outPath := filepath.Join(tmpDir, "admin-chart.tgz")

	output, err := executeCommand(rootCmd,
		"admin-chart ../tests/charts/test-chart-admin --namespace test-ns --output "+outPath)
	require.NoError(t, err)
	assert.Contains(t, output, "CustomResourceDefinition: 2")
	assert.Contains(t, output, "ClusterRole: 2")
	assert.Contains(t, output, "ClusterRoleBinding: 1")
	assert.Contains(t, output, "ValidatingWebhookConfiguration: 1")

	loaded, err := loader.Load(outPath)
	require.NoError(t, err)
	assert.Equal(t, "datarobot-admin", loaded.Metadata.Name)
	assert.Equal(t, "11.10.88", loaded.Metadata.Version)
}

func TestCommandAdminChart_CustomReleaseName(t *testing.T) {
	tmpDir := t.TempDir()
	outPath := filepath.Join(tmpDir, "admin-chart.tgz")

	output, err := executeCommand(rootCmd,
		"admin-chart ../tests/charts/test-chart-admin --namespace test-ns --release-name my-dr --output "+outPath)
	require.NoError(t, err)
	assert.Contains(t, output, "ClusterRole: 2")

	loaded, err := loader.Load(outPath)
	require.NoError(t, err)

	for _, tmpl := range loaded.Templates {
		content := string(tmpl.Data)
		if tmpl.Name == "templates/clusterroles.yaml" {
			assert.Contains(t, content, "my-dr-manager-role")
			assert.Contains(t, content, "my-dr-reader-role")
		}
	}
}

func TestCommandAdminChart_Debug(t *testing.T) {
	tmpDir := t.TempDir()
	outPath := filepath.Join(tmpDir, "admin-chart.tgz")

	output, err := executeCommand(rootCmd,
		"admin-chart ../tests/charts/test-chart-admin --namespace test-ns --debug --output "+outPath)
	require.NoError(t, err)
	assert.Contains(t, output, "testresources.example.datarobot.com")
}

func TestCommandAdminChart_DisabledCRDs(t *testing.T) {
	tmpDir := t.TempDir()
	outPath := filepath.Join(tmpDir, "admin-chart.tgz")

	output, err := executeCommand(rootCmd,
		"admin-chart ../tests/charts/test-chart-admin --namespace test-ns --set installCRDs=false --output "+outPath)
	require.NoError(t, err)
	assert.NotContains(t, output, "CustomResourceDefinition")
	assert.Contains(t, output, "ClusterRole: 2")
}

func TestCommandAdminChart_MissingNamespace(t *testing.T) {
	tmpDir := t.TempDir()
	outPath := filepath.Join(tmpDir, "admin-chart.tgz")

	_, err := executeCommand(rootCmd,
		"admin-chart ../tests/charts/test-chart-admin --output "+outPath)
	assert.Error(t, err)
}

func TestCommandAdminChart_MissingOutput(t *testing.T) {
	_, err := executeCommand(rootCmd,
		"admin-chart ../tests/charts/test-chart-admin --namespace test-ns")
	assert.Error(t, err)
}
