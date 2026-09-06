package manifest

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testManifest = `---
# Source: chart/templates/crd1.yaml
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: widgets.example.com
---
# Source: chart/templates/crd2.yaml
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: gadgets.example.com
---
# Source: chart/templates/clusterrole1.yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: widget-reader
---
# Source: chart/templates/clusterrole2.yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: gadget-reader
---
# Source: chart/templates/clusterrolebinding.yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: widget-reader-binding
---
# Source: chart/templates/webhook.yaml
apiVersion: admissionregistration.k8s.io/v1
kind: ValidatingWebhookConfiguration
metadata:
  name: widget-validator
---
# Source: chart/templates/deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: widget-app
  namespace: production
---
# Source: chart/templates/service.yaml
apiVersion: v1
kind: Service
metadata:
  name: widget-svc
  namespace: production
`

func TestParseManifests(t *testing.T) {
	resources, err := ParseManifests(testManifest)
	require.NoError(t, err)
	assert.Len(t, resources, 8)

	// Verify kinds
	kinds := make([]string, len(resources))
	for i, r := range resources {
		kinds[i] = r.Kind
	}
	assert.Contains(t, kinds, "CustomResourceDefinition")
	assert.Contains(t, kinds, "ClusterRole")
	assert.Contains(t, kinds, "ClusterRoleBinding")
	assert.Contains(t, kinds, "ValidatingWebhookConfiguration")
	assert.Contains(t, kinds, "Deployment")
	assert.Contains(t, kinds, "Service")

	// Verify names and namespaces for namespaced resources
	var deployment, service *Resource
	for i := range resources {
		switch resources[i].Kind {
		case "Deployment":
			deployment = &resources[i]
		case "Service":
			service = &resources[i]
		}
	}
	require.NotNil(t, deployment)
	assert.Equal(t, "widget-app", deployment.Name)
	assert.Equal(t, "production", deployment.Namespace)

	require.NotNil(t, service)
	assert.Equal(t, "widget-svc", service.Name)
	assert.Equal(t, "production", service.Namespace)
}

func TestParseManifests_PopulatesAPIVersion(t *testing.T) {
	resources, err := ParseManifests(testManifest)
	require.NoError(t, err)
	for _, r := range resources {
		assert.NotEmpty(t, r.APIVersion, "resource %s/%s should have APIVersion", r.Kind, r.Name)
	}
}

func TestParseManifests_SkipsEmptyDocuments(t *testing.T) {
	input := `---
---
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: my-config
  namespace: default
`
	resources, err := ParseManifests(input)
	require.NoError(t, err)
	assert.Len(t, resources, 1)
	assert.Equal(t, "ConfigMap", resources[0].Kind)
	assert.Equal(t, "my-config", resources[0].Name)
}

// Fix 1: PEM block contains "-----BEGIN CERTIFICATE-----" which has "---" substring.
// Must NOT be split on it.
func TestParseManifests_PEMBlock(t *testing.T) {
	input := `---
apiVersion: v1
kind: Secret
metadata:
  name: tls-secret
  namespace: default
type: kubernetes.io/tls
stringData:
  tls.crt: |
    -----BEGIN CERTIFICATE-----
    MIIBkTCB+wIJ...
    -----END CERTIFICATE-----
`
	resources, err := ParseManifests(input)
	require.NoError(t, err)
	require.Len(t, resources, 1, "PEM block must not be split as YAML doc separator")
	assert.Equal(t, "Secret", resources[0].Kind)
	assert.Contains(t, resources[0].RawYAML, "-----BEGIN CERTIFICATE-----")
}

// Fix 1: description field containing "---" substring must not cause split.
func TestParseManifests_DescriptionWithDashes(t *testing.T) {
	input := `---
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: foos.example.com
spec:
  group: example.com
  names:
    kind: Foo
  scope: Cluster
  versions:
    - name: v1
      schema:
        openAPIV3Schema:
          description: "a --- b"
          type: object
`
	resources, err := ParseManifests(input)
	require.NoError(t, err)
	require.Len(t, resources, 1, "description containing '---' must not split document")
	assert.Equal(t, "CustomResourceDefinition", resources[0].Kind)
}

// Fix 1: indented "---" inside a block scalar is NOT a doc separator.
func TestParseManifests_IndentedDashesInBlockScalar(t *testing.T) {
	input := `---
apiVersion: v1
kind: ConfigMap
metadata:
  name: embedded-yaml
  namespace: default
data:
  content: |
    key: value
    ---
    another: doc
`
	resources, err := ParseManifests(input)
	require.NoError(t, err)
	require.Len(t, resources, 1, "indented '---' in block scalar must not split document")
	assert.Equal(t, "ConfigMap", resources[0].Kind)
	assert.Contains(t, resources[0].RawYAML, "---")
}

// Fix 2: Classify — extraAdminKinds forces namespaced Role into Admin.
func TestClassify_ExtraAdminKinds(t *testing.T) {
	resources := []Resource{
		{Kind: "Role", APIVersion: "rbac.authorization.k8s.io/v1", Name: "my-role", Namespace: "default"},
		{Kind: "Deployment", APIVersion: "apps/v1", Name: "app", Namespace: "default"},
		{Kind: "ClusterRole", APIVersion: "rbac.authorization.k8s.io/v1", Name: "cr"},
	}
	result := Classify(resources, []string{"Role"})

	adminKinds := kindsOf(result.Admin)
	appKinds := kindsOf(result.App)

	assert.Contains(t, adminKinds, "Role", "Role forced into admin via extraAdminKinds")
	assert.Contains(t, appKinds, "Deployment")
	assert.Contains(t, adminKinds, "ClusterRole")
	assert.Empty(t, result.Warnings)
}

// Fix 2: Classify — case-insensitive extraAdminKinds match.
func TestClassify_ExtraAdminKindsCaseInsensitive(t *testing.T) {
	resources := []Resource{
		{Kind: "ServiceAccount", APIVersion: "v1", Name: "sa", Namespace: "default"},
	}
	result := Classify(resources, []string{"serviceaccount"})
	assert.Len(t, result.Admin, 1)
	assert.Empty(t, result.App)
}

// Fix 2: Classify — CR classified via CRD spec.scope=Cluster -> Admin.
func TestClassify_CRViaCRD_ClusterScoped(t *testing.T) {
	crdYAML := `apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: foos.example.com
spec:
  group: example.com
  names:
    kind: Foo
  scope: Cluster`

	resources := []Resource{
		{Kind: "CustomResourceDefinition", APIVersion: "apiextensions.k8s.io/v1", Name: "foos.example.com", RawYAML: crdYAML},
		{Kind: "Foo", APIVersion: "example.com/v1", Name: "my-foo"},
	}
	result := Classify(resources, nil)

	adminKinds := kindsOf(result.Admin)
	assert.Contains(t, adminKinds, "Foo", "CR with Cluster-scope CRD must go to Admin")
	assert.Contains(t, adminKinds, "CustomResourceDefinition")
	assert.Empty(t, result.Warnings)
}

// Fix 2: Classify — CR classified via CRD spec.scope=Namespaced -> App.
func TestClassify_CRViaCRD_NamespacedScoped(t *testing.T) {
	crdYAML := `apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: bars.mygroup.io
spec:
  group: mygroup.io
  names:
    kind: Bar
  scope: Namespaced`

	resources := []Resource{
		{Kind: "CustomResourceDefinition", APIVersion: "apiextensions.k8s.io/v1", Name: "bars.mygroup.io", RawYAML: crdYAML},
		{Kind: "Bar", APIVersion: "mygroup.io/v1", Name: "my-bar", Namespace: "default"},
	}
	result := Classify(resources, nil)

	appKinds := kindsOf(result.App)
	assert.Contains(t, appKinds, "Bar", "CR with Namespaced CRD must go to App")
	assert.Empty(t, result.Warnings)
}

// Fix 2: Classify — Deployment without namespace goes to App via static table (key regression).
func TestClassify_DeploymentWithoutNamespace_GoesToApp(t *testing.T) {
	resources := []Resource{
		{Kind: "Deployment", APIVersion: "apps/v1", Name: "app"},
	}
	result := Classify(resources, nil)

	assert.Len(t, result.App, 1, "Deployment must go to App even without namespace field")
	assert.Empty(t, result.Admin)
	assert.Empty(t, result.Warnings)
}

// Fix 2: Classify — unknown kind falls back to namespace heuristic + warning.
func TestClassify_UnknownKind_FallbackWarning(t *testing.T) {
	resources := []Resource{
		{Kind: "MyCustomThing", APIVersion: "custom.io/v1", Name: "thing1", Namespace: ""},
		{Kind: "MyCustomThing", APIVersion: "custom.io/v1", Name: "thing2", Namespace: "ns"},
	}
	result := Classify(resources, nil)

	assert.Len(t, result.Admin, 1)
	assert.Len(t, result.App, 1)
	assert.Len(t, result.Warnings, 2)
	assert.Contains(t, result.Warnings[0], "unknown kind MyCustomThing")
	assert.Contains(t, result.Warnings[0], "custom.io/v1")
	assert.Contains(t, result.Warnings[0], "classified as admin")
	assert.Contains(t, result.Warnings[0], "thing1")
	assert.Contains(t, result.Warnings[1], "classified as app")
	assert.Contains(t, result.Warnings[1], "ns/thing2")
}

// Fix 2: Classify — static table cluster-scoped kinds go to Admin.
func TestClassify_StaticClusterScoped(t *testing.T) {
	resources := []Resource{
		{Kind: "Namespace", APIVersion: "v1", Name: "my-ns"},
		{Kind: "ClusterRole", APIVersion: "rbac.authorization.k8s.io/v1", Name: "cr"},
		{Kind: "StorageClass", APIVersion: "storage.k8s.io/v1", Name: "sc"},
	}
	result := Classify(resources, nil)
	assert.Len(t, result.Admin, 3)
	assert.Empty(t, result.App)
	assert.Empty(t, result.Warnings)
}

// Fix 2: Classify — static table namespaced kinds go to App.
func TestClassify_StaticNamespaced(t *testing.T) {
	resources := []Resource{
		{Kind: "ConfigMap", APIVersion: "v1", Name: "cfg"},
		{Kind: "Secret", APIVersion: "v1", Name: "sec"},
		{Kind: "Service", APIVersion: "v1", Name: "svc"},
	}
	result := Classify(resources, nil)
	assert.Len(t, result.App, 3)
	assert.Empty(t, result.Admin)
	assert.Empty(t, result.Warnings)
}

func TestGroupByKind(t *testing.T) {
	resources, err := ParseManifests(testManifest)
	require.NoError(t, err)

	result := Classify(resources, nil)
	groups := GroupByKind(result.Admin)

	assert.Len(t, groups, 4)
	assert.Len(t, groups["CustomResourceDefinition"], 2)
	assert.Len(t, groups["ClusterRole"], 2)
	assert.Len(t, groups["ClusterRoleBinding"], 1)
	assert.Len(t, groups["ValidatingWebhookConfiguration"], 1)
}

func TestSummary(t *testing.T) {
	resources, err := ParseManifests(testManifest)
	require.NoError(t, err)

	result := Classify(resources, nil)
	summary := Summary(result.Admin)

	expected := "ClusterRole: 2, ClusterRoleBinding: 1, CustomResourceDefinition: 2, ValidatingWebhookConfiguration: 1"
	assert.Equal(t, expected, summary)
}

// Fix 3: WithAnnotation sets annotation in RawYAML.
func TestWithAnnotation(t *testing.T) {
	r := Resource{
		Kind:       "Deployment",
		APIVersion: "apps/v1",
		Name:       "app",
		Namespace:  "default",
		RawYAML: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: app
  namespace: default`,
	}

	got, err := r.WithAnnotation("my-key", "my-value")
	require.NoError(t, err)
	assert.Contains(t, got.RawYAML, "my-key")
	assert.Contains(t, got.RawYAML, "my-value")
	// Original unchanged
	assert.NotContains(t, r.RawYAML, "my-key")
}

// Fix 3: WithAnnotation adds to existing annotations.
func TestWithAnnotation_ExistingAnnotations(t *testing.T) {
	r := Resource{
		Kind: "ConfigMap",
		RawYAML: `apiVersion: v1
kind: ConfigMap
metadata:
  name: cfg
  annotations:
    existing-key: existing-value`,
	}

	got, err := r.WithAnnotation("new-key", "new-value")
	require.NoError(t, err)
	assert.Contains(t, got.RawYAML, "existing-key")
	assert.Contains(t, got.RawYAML, "new-key")
}

// Fix 3: StripHelmHookAnnotations removes helm.sh/hook* annotations.
func TestStripHelmHookAnnotations_RemovesHookAnnotations(t *testing.T) {
	r := Resource{
		Kind: "Job",
		RawYAML: `apiVersion: batch/v1
kind: Job
metadata:
  name: migrate
  annotations:
    helm.sh/hook: pre-install
    helm.sh/hook-weight: "0"
    helm.sh/hook-delete-policy: before-hook-creation
    keep-this: value`,
	}

	got, err := r.StripHelmHookAnnotations()
	require.NoError(t, err)
	assert.NotContains(t, got.RawYAML, "helm.sh/hook")
	assert.Contains(t, got.RawYAML, "keep-this")
}

// Fix 3: StripHelmHookAnnotations removes annotations map when empty after strip.
func TestStripHelmHookAnnotations_EmptiesAnnotations(t *testing.T) {
	r := Resource{
		Kind: "Job",
		RawYAML: `apiVersion: batch/v1
kind: Job
metadata:
  name: migrate
  annotations:
    helm.sh/hook: pre-install
    helm.sh/hook-weight: "0"`,
	}

	got, err := r.StripHelmHookAnnotations()
	require.NoError(t, err)
	assert.NotContains(t, got.RawYAML, "helm.sh/hook")
	assert.NotContains(t, got.RawYAML, "annotations", "empty annotations map should be removed")
}

func TestParseManifests_CommentSeparator(t *testing.T) {
	input := `apiVersion: v1
kind: ConfigMap
metadata:
  name: first
--- # this is a comment
apiVersion: v1
kind: ConfigMap
metadata:
  name: second
`
	resources, err := ParseManifests(input)
	require.NoError(t, err)
	assert.Len(t, resources, 2)
	names := []string{resources[0].Name, resources[1].Name}
	assert.Contains(t, names, "first")
	assert.Contains(t, names, "second")
}

func TestParseManifests_DashesWithContent_NotSplit(t *testing.T) {
	input := `apiVersion: v1
kind: ConfigMap
metadata:
  name: cfg
data:
  key: ---foo
`
	resources, err := ParseManifests(input)
	require.NoError(t, err)
	require.Len(t, resources, 1, "---foo is not a doc separator")
}

func TestClassify_MalformedCRD_Warning(t *testing.T) {
	// CRD with spec.scope as a list (type mismatch) — parses as YAML but won't fit minimalCRD.
	// sigs.k8s.io/yaml converts via JSON; a list for a string field will cause unmarshal error.
	malformedCRD := Resource{
		Kind:       "CustomResourceDefinition",
		APIVersion: "apiextensions.k8s.io/v1",
		Name:       "bad-crd.example.com",
		RawYAML: `apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: bad-crd.example.com
spec:
  group: example.com
  names:
    kind: Bad
  scope:
    - invalid
    - list`,
	}
	result := Classify([]Resource{malformedCRD}, nil)
	require.Len(t, result.Warnings, 1)
	assert.Contains(t, result.Warnings[0], "failed to parse CRD bad-crd.example.com")
}

// kindsOf extracts Kind values from a resource slice.
func kindsOf(resources []Resource) []string {
	out := make([]string, len(resources))
	for i, r := range resources {
		out[i] = r.Kind
	}
	return out
}

// Ensure fast path returns identical string (not a round-tripped copy).
func TestStripHelmHookAnnotations_FastPathReturnsSameString(t *testing.T) {
	rawYAML := `apiVersion: v1
kind: ConfigMap
metadata:
  name: cfg
  annotations:
    some-other-annotation: value`
	r := Resource{Kind: "ConfigMap", RawYAML: rawYAML}
	got, err := r.StripHelmHookAnnotations()
	require.NoError(t, err)
	// No helm.sh/hook present → fast path → identical pointer/value.
	assert.Equal(t, rawYAML, got.RawYAML)
	assert.False(t, strings.Contains(got.RawYAML, "helm.sh/hook"))
}
