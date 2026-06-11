package manifest

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const roleManifestBase = `---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: role-a
  namespace: default
rules:
  - apiGroups: [""]
    resources: ["pods"]
    verbs: ["get", "list"]
  - apiGroups: ["apps"]
    resources: ["deployments"]
    verbs: ["get"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: role-b
  namespace: default
rules:
  - apiGroups: [""]
    resources: ["configmaps"]
    verbs: ["get"]
  - apiGroups: ["apps"]
    resources: ["deployments"]
    verbs: ["get"]
`

func TestRoleRules_Union(t *testing.T) {
	resources, err := ParseManifests(roleManifestBase)
	require.NoError(t, err)

	rules, err := RoleRules(resources)
	require.NoError(t, err)

	// role-a: 2 rules, role-b: 2 rules, but "apps/deployments get" is duplicate → 3 unique
	assert.Len(t, rules, 3)

	// first-seen order: pods, deployments, configmaps
	assert.Equal(t, []interface{}{"pods"}, rules[0]["resources"])
	assert.Equal(t, []interface{}{"deployments"}, rules[1]["resources"])
	assert.Equal(t, []interface{}{"configmaps"}, rules[2]["resources"])
}

func TestRoleRules_EmptyAndMissingRulesSkipped(t *testing.T) {
	manifest := `---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: role-empty
  namespace: default
rules: []
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: role-no-rules
  namespace: default
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: role-real
  namespace: default
rules:
  - apiGroups: [""]
    resources: ["secrets"]
    verbs: ["get"]
`
	resources, err := ParseManifests(manifest)
	require.NoError(t, err)

	rules, err := RoleRules(resources)
	require.NoError(t, err)
	assert.Len(t, rules, 1)
	assert.Equal(t, []interface{}{"secrets"}, rules[0]["resources"])
}

func TestRoleRules_ResourceNamesPreserved(t *testing.T) {
	manifest := `---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: role-named
  namespace: default
rules:
  - apiGroups: [""]
    resources: ["configmaps"]
    resourceNames: ["my-config", "other-config"]
    verbs: ["get"]
`
	resources, err := ParseManifests(manifest)
	require.NoError(t, err)

	rules, err := RoleRules(resources)
	require.NoError(t, err)
	require.Len(t, rules, 1)

	rn, ok := rules[0]["resourceNames"]
	require.True(t, ok, "resourceNames key missing")
	assert.Equal(t, []interface{}{"my-config", "other-config"}, rn)
}

func TestRoleRules_WildcardApiGroupPreserved(t *testing.T) {
	manifest := `---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: role-wildcard
  namespace: default
rules:
  - apiGroups: ["*"]
    resources: ["*"]
    verbs: ["*"]
`
	resources, err := ParseManifests(manifest)
	require.NoError(t, err)

	rules, err := RoleRules(resources)
	require.NoError(t, err)
	require.Len(t, rules, 1)
	assert.Equal(t, []interface{}{"*"}, rules[0]["apiGroups"])
	assert.Equal(t, []interface{}{"*"}, rules[0]["resources"])
	assert.Equal(t, []interface{}{"*"}, rules[0]["verbs"])
}

func TestRoleRules_DuplicateRuleAppearsOnce(t *testing.T) {
	manifest := `---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: role-x
  namespace: default
rules:
  - apiGroups: [""]
    resources: ["pods"]
    verbs: ["get"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: role-y
  namespace: default
rules:
  - apiGroups: [""]
    resources: ["pods"]
    verbs: ["get"]
`
	resources, err := ParseManifests(manifest)
	require.NoError(t, err)

	rules, err := RoleRules(resources)
	require.NoError(t, err)
	assert.Len(t, rules, 1)
}

func TestRoleRules_NonRoleKindsIgnored(t *testing.T) {
	manifest := `---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: cluster-admin-role
rules:
  - apiGroups: ["*"]
    resources: ["*"]
    verbs: ["*"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: binding
  namespace: default
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: app
  namespace: default
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: role-only
  namespace: default
rules:
  - apiGroups: [""]
    resources: ["pods"]
    verbs: ["list"]
`
	resources, err := ParseManifests(manifest)
	require.NoError(t, err)

	rules, err := RoleRules(resources)
	require.NoError(t, err)
	// Only role-only's rule; ClusterRole rules ignored
	require.Len(t, rules, 1)
	assert.Equal(t, []interface{}{"pods"}, rules[0]["resources"])
}

func TestRoleRules_MalformedRoleRulesError(t *testing.T) {
	// Build a Resource manually with malformed rules field
	r := Resource{
		Kind:       "Role",
		APIVersion: "rbac.authorization.k8s.io/v1",
		Name:       "bad-role",
		Namespace:  "default",
		RawYAML: `apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: bad-role
  namespace: default
rules:
  - "not-a-map"
`,
	}

	_, err := RoleRules([]Resource{r})
	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "bad-role"),
		"error should mention role name, got: %s", err.Error())
}

func TestRoleRules_EmptyInput(t *testing.T) {
	rules, err := RoleRules(nil)
	require.NoError(t, err)
	assert.Nil(t, rules)
}

// TestRoleRules_CRDDefinedRoleIgnored ensures that a CRD-defined kind named
// "Role" with a non-rbac apiVersion (e.g. rabbitmq.com/v1) is silently ignored
// and does not inject garbage rules into the union.
func TestRoleRules_CRDDefinedRoleIgnored(t *testing.T) {
	manifest := `---
apiVersion: rabbitmq.com/v1
kind: Role
metadata:
  name: rabbitmq-role
  namespace: default
rules:
  - apiGroups: ["*"]
    resources: ["*"]
    verbs: ["*"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: real-role
  namespace: default
rules:
  - apiGroups: [""]
    resources: ["pods"]
    verbs: ["get"]
`
	resources, err := ParseManifests(manifest)
	require.NoError(t, err)

	rules, err := RoleRules(resources)
	require.NoError(t, err)
	// Only real-role's rule; rabbitmq Role ignored
	require.Len(t, rules, 1)
	assert.Equal(t, []interface{}{"pods"}, rules[0]["resources"])
}
