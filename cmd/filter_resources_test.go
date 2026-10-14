package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/datarobot-oss/helm-datarobot-plugin/pkg/manifest"
	"github.com/mattn/go-shellwords"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// executeCommandWithInput mirrors executeCommand (root_test.go) but also feeds stdin,
// which filter-resources reads via cmd.InOrStdin().
func executeCommandWithInput(root *cobra.Command, cmd string, stdin string) (output string, err error) {
	buf := new(bytes.Buffer)

	args, err := shellwords.Parse(cmd)
	if err != nil {
		return "", err
	}
	resetSubCommandFlagValues(root) // See: https://github.com/spf13/cobra/issues/1488
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetIn(strings.NewReader(stdin))
	root.SetArgs(args)

	err = root.Execute()
	return strings.TrimSpace(buf.String()), err
}

const sampleManifest = `apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: widgets.example.com
spec:
  group: example.com
  scope: Namespaced
  names:
    kind: Widget
    plural: widgets
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: api
  namespace: dr
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: some-cluster-role
`

func runFilter(t *testing.T, cmd string) string {
	t.Helper()
	out, err := executeCommandWithInput(rootCmd, cmd, sampleManifest)
	require.NoError(t, err)
	return out
}

func TestFilterResourcesKeepApp(t *testing.T) {
	out := runFilter(t, "filter-resources --keep app")
	assert.Contains(t, out, "kind: Deployment")
	assert.NotContains(t, out, "kind: ClusterRole")
	assert.NotContains(t, out, "kind: CustomResourceDefinition")
}

func TestFilterResourcesKeepAdmin(t *testing.T) {
	out := runFilter(t, "filter-resources --keep admin")
	assert.Contains(t, out, "kind: ClusterRole")
	assert.Contains(t, out, "kind: CustomResourceDefinition")
	assert.NotContains(t, out, "kind: Deployment")
}

func TestFilterResourcesDefaultKeepIsApp(t *testing.T) {
	out := runFilter(t, "filter-resources")
	assert.Contains(t, out, "kind: Deployment")
	assert.NotContains(t, out, "kind: ClusterRole")
}

func TestFilterResourcesInvalidKeep(t *testing.T) {
	_, err := executeCommandWithInput(rootCmd, "filter-resources --keep bogus", sampleManifest)
	assert.Error(t, err)
}

func TestFilterResourcesEmptyResult(t *testing.T) {
	// Only namespaced resources — keep=admin yields an empty partition, must exit 0 with no output.
	input := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: api
  namespace: dr
`
	out, err := executeCommandWithInput(rootCmd, "filter-resources --keep admin", input)
	require.NoError(t, err)
	assert.Equal(t, "", out)
}

func TestFilterResourcesExtraAdminKinds(t *testing.T) {
	out := runFilter(t, "filter-resources --keep admin --extra-admin-kinds Deployment")
	assert.Contains(t, out, "kind: Deployment")
	assert.Contains(t, out, "kind: ClusterRole")
}

// Partition invariant (spec §10, p1test-a2): Admin ∪ App == all, Admin ∩ App == ∅.
// manifest.Classify is the single source of truth shared by the infra-chart builder
// and this filter post-renderer — the two halves must always be complementary.
func TestClassifyPartitionInvariant(t *testing.T) {
	resources, err := manifest.ParseManifests(sampleManifest)
	assert.NoError(t, err)

	res := manifest.Classify(resources, nil)

	// Union-completeness: every parsed resource lands in exactly one partition.
	assert.Equal(t, len(resources), len(res.Admin)+len(res.App), "Admin ∪ App must equal all")

	// Disjointness: no resource appears in both partitions.
	seen := map[string]bool{}
	for _, r := range append(append([]manifest.Resource{}, res.Admin...), res.App...) {
		key := r.Kind + "/" + r.Namespace + "/" + r.Name
		assert.False(t, seen[key], "resource %s appeared in both partitions", key)
		seen[key] = true
	}
}
