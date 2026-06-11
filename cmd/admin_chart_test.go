package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/datarobot-oss/helm-datarobot-plugin/pkg/render_helper"
	"github.com/mattn/go-shellwords"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	helmchart "helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	sigsyaml "sigs.k8s.io/yaml"
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

// findTemplate returns the *helmchart.File with the given name, or nil.
func findTemplate(templates []*helmchart.File, name string) *helmchart.File {
	for _, tmpl := range templates {
		if tmpl.Name == name {
			return tmpl
		}
	}
	return nil
}

// parseMultiDoc splits a YAML multi-doc string and unmarshals each doc into a map.
func parseMultiDoc(t *testing.T, data string) []map[string]interface{} {
	t.Helper()
	var docs []map[string]interface{}
	for _, part := range strings.Split(data, "\n---") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		var m map[string]interface{}
		require.NoError(t, sigsyaml.Unmarshal([]byte(part), &m))
		if len(m) > 0 {
			docs = append(docs, m)
		}
	}
	return docs
}

func TestCommandAdminChart_PipelineSA(t *testing.T) {
	tmpDir := t.TempDir()
	outPath := filepath.Join(tmpDir, "admin-chart.tgz")

	_, err := executeCommand(rootCmd,
		"admin-chart ../tests/charts/test-chart-admin --namespace test-ns --pipeline-sa pipeline --output "+outPath)
	require.NoError(t, err)

	loaded, err := loader.Load(outPath)
	require.NoError(t, err)

	// pipeline-rbac.yaml must be present
	rbacFile := findTemplate(loaded.Templates, "templates/pipeline-rbac.yaml")
	require.NotNil(t, rbacFile, "templates/pipeline-rbac.yaml must exist when --pipeline-sa set")

	docs := parseMultiDoc(t, string(rbacFile.Data))

	// Collect by kind+name for easy assertion
	type kindName struct{ kind, name string }
	byKN := make(map[kindName]map[string]interface{})
	for _, d := range docs {
		k, _ := d["kind"].(string)
		meta, _ := d["metadata"].(map[string]interface{})
		n, _ := meta["name"].(string)
		byKN[kindName{k, n}] = d
	}

	// ServiceAccount "pipeline"
	sa, ok := byKN[kindName{"ServiceAccount", "pipeline"}]
	require.True(t, ok, "ServiceAccount 'pipeline' must exist")
	meta := sa["metadata"].(map[string]interface{})
	assert.Equal(t, "test-ns", meta["namespace"])

	// RoleBinding datarobot-pipeline-admin → ClusterRole admin
	rb, ok := byKN[kindName{"RoleBinding", "datarobot-pipeline-admin"}]
	require.True(t, ok, "RoleBinding 'datarobot-pipeline-admin' must exist")
	roleRef := rb["roleRef"].(map[string]interface{})
	assert.Equal(t, "ClusterRole", roleRef["kind"])
	assert.Equal(t, "admin", roleRef["name"])

	// ClusterRole datarobot-pipeline-cluster-read
	_, ok = byKN[kindName{"ClusterRole", "datarobot-pipeline-cluster-read"}]
	assert.True(t, ok, "ClusterRole 'datarobot-pipeline-cluster-read' must exist")

	// ClusterRoleBinding datarobot-pipeline-cluster-read
	_, ok = byKN[kindName{"ClusterRoleBinding", "datarobot-pipeline-cluster-read"}]
	assert.True(t, ok, "ClusterRoleBinding 'datarobot-pipeline-cluster-read' must exist")

	// ClusterRole datarobot-pipeline-role-union — present because fixture has role-with-resourcenames
	crUnion, ok := byKN[kindName{"ClusterRole", "datarobot-pipeline-role-union"}]
	require.True(t, ok, "ClusterRole 'datarobot-pipeline-role-union' must exist")

	// union rules must contain the resourceNames rule
	rules, _ := crUnion["rules"].([]interface{})
	require.NotEmpty(t, rules, "union ClusterRole must have rules")

	foundResourceNames := false
	foundWildcardApiGroups := false
	for _, ri := range rules {
		rule, _ := ri.(map[string]interface{})
		if rn, ok := rule["resourceNames"]; ok {
			rnSlice, _ := rn.([]interface{})
			if len(rnSlice) > 0 {
				foundResourceNames = true
			}
		}
		if ag, ok := rule["apiGroups"]; ok {
			agSlice, _ := ag.([]interface{})
			for _, a := range agSlice {
				if a == "*" {
					foundWildcardApiGroups = true
				}
			}
		}
	}
	assert.True(t, foundResourceNames, "union ClusterRole must include rule with resourceNames")
	assert.True(t, foundWildcardApiGroups, "union ClusterRole must include rule with wildcard apiGroups")

	// empty-rules Role must NOT contribute any rules (rules from role-empty-rules are skipped)
	// Verify by checking no rule references an empty resource list originating from that Role —
	// specifically the union must have exactly 2 rules (both from role-with-resourcenames).
	assert.Len(t, rules, 2, "union must have exactly 2 rules (role-empty-rules contributes 0)")

	// RoleBinding datarobot-pipeline-role-union
	_, ok = byKN[kindName{"RoleBinding", "datarobot-pipeline-role-union"}]
	assert.True(t, ok, "RoleBinding 'datarobot-pipeline-role-union' must exist")
}

func TestCommandAdminChart_PipelineSA_CollisionWarning(t *testing.T) {
	tmpDir := t.TempDir()
	outPath := filepath.Join(tmpDir, "admin-chart.tgz")

	// The test chart renders a ServiceAccount named "datarobot-sa" (Release.Name + "-sa").
	// Pass --extra-admin-kinds=ServiceAccount to move it to the admin partition, then
	// use --pipeline-sa=datarobot-sa to trigger the collision warning.
	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	args, parseErr := shellwords.Parse(
		"admin-chart ../tests/charts/test-chart-admin --namespace test-ns --extra-admin-kinds=ServiceAccount --pipeline-sa=datarobot-sa --output " + outPath)
	require.NoError(t, parseErr)
	resetSubCommandFlagValues(rootCmd)
	rootCmd.SetOut(outBuf)
	rootCmd.SetErr(errBuf)
	rootCmd.SetArgs(args)
	execErr := rootCmd.Execute()
	require.NoError(t, execErr)
	assert.Contains(t, errBuf.String(), "collides", "expected SA collision warning on stderr")
}

func TestCommandAdminChart_NoPipelineSA(t *testing.T) {
	tmpDir := t.TempDir()
	outPath := filepath.Join(tmpDir, "admin-chart.tgz")

	_, err := executeCommand(rootCmd,
		"admin-chart ../tests/charts/test-chart-admin --namespace test-ns --output "+outPath)
	require.NoError(t, err)

	loaded, err := loader.Load(outPath)
	require.NoError(t, err)

	// pipeline-rbac.yaml must NOT be present without --pipeline-sa
	rbacFile := findTemplate(loaded.Templates, "templates/pipeline-rbac.yaml")
	assert.Nil(t, rbacFile, "templates/pipeline-rbac.yaml must NOT exist when --pipeline-sa is not set")
}

func TestCommandAdminChart_PipelineSA_CustomClusterReadKinds(t *testing.T) {
	tmpDir := t.TempDir()
	outPath := filepath.Join(tmpDir, "admin-chart.tgz")

	_, err := executeCommand(rootCmd,
		"admin-chart ../tests/charts/test-chart-admin --namespace test-ns --pipeline-sa pipeline --cluster-read-kinds=pods,configmaps.v1 --output "+outPath)
	require.NoError(t, err)

	loaded, err := loader.Load(outPath)
	require.NoError(t, err)

	rbacFile := findTemplate(loaded.Templates, "templates/pipeline-rbac.yaml")
	require.NotNil(t, rbacFile)

	docs := parseMultiDoc(t, string(rbacFile.Data))

	// Find ClusterRole datarobot-pipeline-cluster-read
	var clusterReadDoc map[string]interface{}
	for _, d := range docs {
		if d["kind"] == "ClusterRole" {
			meta, _ := d["metadata"].(map[string]interface{})
			if meta["name"] == "datarobot-pipeline-cluster-read" {
				clusterReadDoc = d
				break
			}
		}
	}
	require.NotNil(t, clusterReadDoc, "ClusterRole datarobot-pipeline-cluster-read must exist")

	rulesRaw, _ := clusterReadDoc["rules"].([]interface{})
	require.NotEmpty(t, rulesRaw, "cluster-read ClusterRole must have rules")

	// Assert a rule with resources=[configmaps] and apiGroups=[v1] (from input configmaps.v1)
	foundConfigmapsRule := false
	foundPodsRule := false
	for _, ri := range rulesRaw {
		rule, _ := ri.(map[string]interface{})
		resourcesRaw, _ := rule["resources"].([]interface{})
		apiGroupsRaw, _ := rule["apiGroups"].([]interface{})
		resources := make([]string, len(resourcesRaw))
		for i, r := range resourcesRaw {
			resources[i], _ = r.(string)
		}
		apiGroups := make([]string, len(apiGroupsRaw))
		for i, g := range apiGroupsRaw {
			apiGroups[i], _ = g.(string)
		}
		if len(resources) == 1 && resources[0] == "configmaps" && len(apiGroups) == 1 && apiGroups[0] == "v1" {
			foundConfigmapsRule = true
		}
		if len(resources) == 1 && resources[0] == "pods" {
			foundPodsRule = true
		}
	}
	assert.True(t, foundConfigmapsRule, "cluster-read ClusterRole must have rule with resources=[configmaps] and apiGroups=[v1]")
	assert.True(t, foundPodsRule, "cluster-read ClusterRole must have rule with resources=[pods]")

	// Default kind (storageclasses) must NOT appear
	assert.NotContains(t, string(rbacFile.Data), "storageclasses")
}
