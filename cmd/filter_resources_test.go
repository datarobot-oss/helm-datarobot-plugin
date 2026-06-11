package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const filterTestInput = `apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-app
  namespace: default
spec:
  replicas: 1
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: my-cluster-role
rules: []
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: my-sa
  namespace: default
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: my-crb
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: my-cluster-role
subjects: []`

func TestCommandFilterResources_KeepApp(t *testing.T) {
	buf := new(bytes.Buffer)
	resetSubCommandFlagValues(rootCmd)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(new(bytes.Buffer))
	rootCmd.SetIn(strings.NewReader(filterTestInput))
	rootCmd.SetArgs([]string{"filter-resources", "--keep=app"})

	err := rootCmd.Execute()
	require.NoError(t, err)

	out := buf.String()
	assert.Contains(t, out, "Deployment")
	assert.Contains(t, out, "ServiceAccount")
	assert.NotContains(t, out, "ClusterRole")
	assert.NotContains(t, out, "ClusterRoleBinding")
}

func TestCommandFilterResources_KeepAdmin(t *testing.T) {
	buf := new(bytes.Buffer)
	resetSubCommandFlagValues(rootCmd)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(new(bytes.Buffer))
	rootCmd.SetIn(strings.NewReader(filterTestInput))
	rootCmd.SetArgs([]string{"filter-resources", "--keep=admin"})

	err := rootCmd.Execute()
	require.NoError(t, err)

	out := buf.String()
	assert.Contains(t, out, "ClusterRole")
	assert.Contains(t, out, "ClusterRoleBinding")
	assert.NotContains(t, out, "Deployment")
	assert.NotContains(t, out, "ServiceAccount")
}

func TestCommandFilterResources_InvalidKeep(t *testing.T) {
	resetSubCommandFlagValues(rootCmd)
	rootCmd.SetOut(new(bytes.Buffer))
	rootCmd.SetErr(new(bytes.Buffer))
	rootCmd.SetIn(strings.NewReader(filterTestInput))
	rootCmd.SetArgs([]string{"filter-resources", "--keep=invalid"})

	err := rootCmd.Execute()
	assert.Error(t, err)
}

func TestCommandFilterResources_EmptyResult(t *testing.T) {
	// Only namespaced resources — keep=admin yields empty result, must exit 0
	input := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-app
  namespace: default`

	buf := new(bytes.Buffer)
	resetSubCommandFlagValues(rootCmd)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(new(bytes.Buffer))
	rootCmd.SetIn(strings.NewReader(input))
	rootCmd.SetArgs([]string{"filter-resources", "--keep=admin"})

	err := rootCmd.Execute()
	require.NoError(t, err, "empty partition must exit 0")
	assert.Equal(t, "", strings.TrimSpace(buf.String()))
}

func TestCommandFilterResources_ExtraAdminKinds(t *testing.T) {
	buf := new(bytes.Buffer)
	resetSubCommandFlagValues(rootCmd)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(new(bytes.Buffer))
	rootCmd.SetIn(strings.NewReader(filterTestInput))
	rootCmd.SetArgs([]string{"filter-resources", "--keep=admin", "--extra-admin-kinds=ServiceAccount"})

	err := rootCmd.Execute()
	require.NoError(t, err)

	out := buf.String()
	assert.Contains(t, out, "ServiceAccount")
	assert.Contains(t, out, "ClusterRole")
}

func TestCommandFilterResources_DefaultKeepIsApp(t *testing.T) {
	buf := new(bytes.Buffer)
	resetSubCommandFlagValues(rootCmd)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(new(bytes.Buffer))
	rootCmd.SetIn(strings.NewReader(filterTestInput))
	rootCmd.SetArgs([]string{"filter-resources"})

	err := rootCmd.Execute()
	require.NoError(t, err)

	out := buf.String()
	assert.Contains(t, out, "Deployment")
	assert.NotContains(t, out, "ClusterRole")
}
