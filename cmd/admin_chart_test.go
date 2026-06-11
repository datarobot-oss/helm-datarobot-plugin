package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/datarobot-oss/helm-datarobot-plugin/pkg/render_helper"
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
	// CRDs: TestResource, TestCluster, BraceTest = 3
	assert.Contains(t, output, "CustomResourceDefinition: 3")
	// ClusterRoles: manager-role, reader-role + hook pre-install-role = 3
	assert.Contains(t, output, "ClusterRole: 3")
	assert.Contains(t, output, "ClusterRoleBinding: 1")
	assert.Contains(t, output, "ValidatingWebhookConfiguration: 1")
	// TestCluster (cluster-scoped CR)
	assert.Contains(t, output, "TestCluster: 1")
	// skipped message present
	assert.Contains(t, output, "skipped")

	loaded, err := loader.Load(outPath)
	require.NoError(t, err)
	assert.Equal(t, "datarobot-admin", loaded.Metadata.Name)
	assert.Equal(t, "11.10.88", loaded.Metadata.Version)

	// Deployment must NOT be in admin chart
	for _, tmpl := range loaded.Templates {
		assert.NotContains(t, tmpl.Name, "deployment")
	}

	// hook annotation must be stripped from pre-install-role
	for _, tmpl := range loaded.Templates {
		if tmpl.Name == "templates/clusterroles.yaml" {
			assert.NotContains(t, string(tmpl.Data), "helm.sh/hook")
			assert.Contains(t, string(tmpl.Data), "pre-install-role")
		}
	}

	// resource-policy: keep must be present on CRDs (default --keep-crds=true)
	for _, tmpl := range loaded.Templates {
		if tmpl.Name == "templates/customresourcedefinitions.yaml" {
			assert.Contains(t, string(tmpl.Data), "helm.sh/resource-policy")
			assert.Contains(t, string(tmpl.Data), "keep")
		}
	}
}

func TestCommandAdminChart_CustomReleaseName(t *testing.T) {
	tmpDir := t.TempDir()
	outPath := filepath.Join(tmpDir, "admin-chart.tgz")

	output, err := executeCommand(rootCmd,
		"admin-chart ../tests/charts/test-chart-admin --namespace test-ns --release-name my-dr --output "+outPath)
	require.NoError(t, err)
	assert.Contains(t, output, "ClusterRole: 3")

	loaded, err := loader.Load(outPath)
	require.NoError(t, err)

	for _, tmpl := range loaded.Templates {
		if tmpl.Name == "templates/clusterroles.yaml" {
			content := string(tmpl.Data)
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
	// crd-with-braces.yaml always renders (no condition), so BraceTest CRD still present
	assert.Contains(t, output, "CustomResourceDefinition: 1")
	assert.Contains(t, output, "ClusterRole: 3")
}

func TestCommandAdminChart_NoNamespaceDeployment(t *testing.T) {
	tmpDir := t.TempDir()
	outPath := filepath.Join(tmpDir, "admin-chart.tgz")

	_, err := executeCommand(rootCmd,
		"admin-chart ../tests/charts/test-chart-admin --namespace test-ns --output "+outPath)
	require.NoError(t, err)

	loaded, err := loader.Load(outPath)
	require.NoError(t, err)

	// deployment-no-ns.yaml Deployment must not appear in admin chart
	for _, tmpl := range loaded.Templates {
		assert.NotEqual(t, "templates/deployments.yaml", tmpl.Name,
			"Deployment (even without namespace) must be in app partition, not admin")
	}
}

func TestCommandAdminChart_DefaultOutput(t *testing.T) {
	tmpDir := t.TempDir()
	absChartPath, err := filepath.Abs("../tests/charts/test-chart-admin")
	require.NoError(t, err)

	origDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(origDir)
	require.NoError(t, os.Chdir(tmpDir))

	output, err := executeCommand(rootCmd,
		"admin-chart "+absChartPath+" --namespace test-ns")
	require.NoError(t, err)
	assert.Contains(t, output, "datarobot-admin-11.10.88.tgz")

	// File must exist
	_, statErr := os.Stat(filepath.Join(tmpDir, "datarobot-admin-11.10.88.tgz"))
	assert.NoError(t, statErr)
}

func TestCommandAdminChart_KeepCRDs(t *testing.T) {
	tmpDir := t.TempDir()
	outPath := filepath.Join(tmpDir, "admin-chart.tgz")

	_, err := executeCommand(rootCmd,
		"admin-chart ../tests/charts/test-chart-admin --namespace test-ns --keep-crds=true --output "+outPath)
	require.NoError(t, err)

	loaded, err := loader.Load(outPath)
	require.NoError(t, err)

	for _, tmpl := range loaded.Templates {
		if strings.Contains(tmpl.Name, "customresourcedefinition") {
			assert.Contains(t, string(tmpl.Data), "helm.sh/resource-policy")
			assert.Contains(t, string(tmpl.Data), "keep")
		}
	}
}

func TestCommandAdminChart_NoKeepCRDs(t *testing.T) {
	tmpDir := t.TempDir()
	outPath := filepath.Join(tmpDir, "admin-chart.tgz")

	_, err := executeCommand(rootCmd,
		"admin-chart ../tests/charts/test-chart-admin --namespace test-ns --keep-crds=false --output "+outPath)
	require.NoError(t, err)

	loaded, err := loader.Load(outPath)
	require.NoError(t, err)

	for _, tmpl := range loaded.Templates {
		if strings.Contains(tmpl.Name, "customresourcedefinition") {
			assert.NotContains(t, string(tmpl.Data), "resource-policy")
		}
	}
}

func TestCommandAdminChart_ExtraAdminKinds(t *testing.T) {
	tmpDir := t.TempDir()
	outPath := filepath.Join(tmpDir, "admin-chart.tgz")

	_, err := executeCommand(rootCmd,
		"admin-chart ../tests/charts/test-chart-admin --namespace test-ns --extra-admin-kinds=ServiceAccount --output "+outPath)
	require.NoError(t, err)

	loaded, err := loader.Load(outPath)
	require.NoError(t, err)

	found := false
	for _, tmpl := range loaded.Templates {
		if tmpl.Name == "templates/serviceaccounts.yaml" {
			found = true
			assert.Contains(t, string(tmpl.Data), "-sa")
		}
	}
	assert.True(t, found, "ServiceAccount must appear in admin chart when --extra-admin-kinds=ServiceAccount")
}

func TestCommandAdminChart_RenderBackRegression(t *testing.T) {
	// Build admin chart from test-chart-admin.
	tmpDir := t.TempDir()
	tgzPath := filepath.Join(tmpDir, "admin.tgz")

	_, err := executeCommand(rootCmd,
		"admin-chart ../tests/charts/test-chart-admin --namespace test-ns --output "+tgzPath)
	require.NoError(t, err)

	// Render the generated admin chart via RenderChart.
	rendered, err := render_helper.RenderChart(tgzPath, nil, nil, nil)
	require.NoError(t, err)

	// Brace-escape fixture: description contains '{{ .Values.something }}' text.
	// After escaping in BuildChart and rendering back, the literal text must survive.
	assert.Contains(t, rendered, "{{ .Values.something }}",
		"brace-escaped Go-template text must round-trip correctly")

	// No backtick-escape residue.
	assert.NotContains(t, rendered, "`{{`",
		"no backtick-escape artifacts in rendered output")

	// No helm.sh/hook annotations.
	assert.NotContains(t, rendered, "helm.sh/hook",
		"hook annotations must be stripped")
}

func TestCommandAdminChart_HookStripped(t *testing.T) {
	tmpDir := t.TempDir()
	outPath := filepath.Join(tmpDir, "admin-chart.tgz")

	_, err := executeCommand(rootCmd,
		"admin-chart ../tests/charts/test-chart-admin --namespace test-ns --output "+outPath)
	require.NoError(t, err)

	loaded, err := loader.Load(outPath)
	require.NoError(t, err)

	for _, tmpl := range loaded.Templates {
		assert.NotContains(t, string(tmpl.Data), "helm.sh/hook:",
			"hook annotations must be stripped from all templates")
	}
}
