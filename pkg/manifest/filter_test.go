package manifest

import (
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

func TestFilterClusterScoped(t *testing.T) {
	resources, err := ParseManifests(testManifest)
	require.NoError(t, err)

	clusterScoped, namespaced := FilterClusterScoped(resources)
	assert.Len(t, clusterScoped, 6)
	assert.Len(t, namespaced, 2)

	for _, r := range clusterScoped {
		assert.Empty(t, r.Namespace, "cluster-scoped resource %s/%s should have no namespace", r.Kind, r.Name)
	}
	for _, r := range namespaced {
		assert.NotEmpty(t, r.Namespace, "namespaced resource %s/%s should have namespace", r.Kind, r.Name)
	}
}

func TestFilterClusterScoped_EmptyNamespaceField(t *testing.T) {
	resources := []Resource{
		{Kind: "ClusterRole", Name: "admin", Namespace: ""},
		{Kind: "ConfigMap", Name: "cfg", Namespace: "default"},
		{Kind: "PersistentVolume", Name: "pv1", Namespace: ""},
	}

	clusterScoped, namespaced := FilterClusterScoped(resources)
	assert.Len(t, clusterScoped, 2)
	assert.Len(t, namespaced, 1)
	assert.Equal(t, "ClusterRole", clusterScoped[0].Kind)
	assert.Equal(t, "PersistentVolume", clusterScoped[1].Kind)
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

func TestGroupByKind(t *testing.T) {
	resources, err := ParseManifests(testManifest)
	require.NoError(t, err)

	clusterScoped, _ := FilterClusterScoped(resources)
	groups := GroupByKind(clusterScoped)

	assert.Len(t, groups, 4)
	assert.Len(t, groups["CustomResourceDefinition"], 2)
	assert.Len(t, groups["ClusterRole"], 2)
	assert.Len(t, groups["ClusterRoleBinding"], 1)
	assert.Len(t, groups["ValidatingWebhookConfiguration"], 1)
}

func TestSummary(t *testing.T) {
	resources, err := ParseManifests(testManifest)
	require.NoError(t, err)

	clusterScoped, _ := FilterClusterScoped(resources)
	summary := Summary(clusterScoped)

	// Sorted alphabetically
	expected := "ClusterRole: 2, ClusterRoleBinding: 1, CustomResourceDefinition: 2, ValidatingWebhookConfiguration: 1"
	assert.Equal(t, expected, summary)
}
