package manifest

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// CRGroups tests
// ---------------------------------------------------------------------------

func TestCRGroups_ThirdPartyOnly(t *testing.T) {
	resources := []Resource{
		// core ("v1") — no slash → group "" → skip
		{Kind: "ConfigMap", APIVersion: "v1", Name: "cm", Namespace: "ns"},
		// legacy built-in: apps → in legacyBuiltinGroups → skip
		{Kind: "Deployment", APIVersion: "apps/v1", Name: "app", Namespace: "ns"},
		// legacy built-in: batch → in legacyBuiltinGroups → skip
		{Kind: "Job", APIVersion: "batch/v1", Name: "job", Namespace: "ns"},
		// networking.k8s.io → ends .k8s.io → skip
		{Kind: "NetworkPolicy", APIVersion: "networking.k8s.io/v1", Name: "np", Namespace: "ns"},
		// metrics.k8s.io → ends .k8s.io → skip
		{Kind: "SomeMetric", APIVersion: "metrics.k8s.io/v1beta1", Name: "m", Namespace: "ns"},
		// cert-manager.io → third-party → included
		{Kind: "Certificate", APIVersion: "cert-manager.io/v1", Name: "cert", Namespace: "ns"},
		// monitoring.coreos.com → third-party → included
		{Kind: "ServiceMonitor", APIVersion: "monitoring.coreos.com/v1", Name: "sm", Namespace: "ns"},
		// datarobot.com → third-party → included
		{Kind: "DataRobotApp", APIVersion: "datarobot.com/v1alpha1", Name: "dr", Namespace: "ns"},
		// duplicate cert-manager.io → dedup
		{Kind: "Issuer", APIVersion: "cert-manager.io/v1", Name: "issuer", Namespace: "ns"},
	}

	// All resources go to App (no extraAdminKinds, none cluster-scoped static).
	result := Classify(resources, nil)

	// Expected CRGroups: cert-manager.io, datarobot.com, monitoring.coreos.com (sorted).
	// apps and batch excluded via legacyBuiltinGroups skip-list.
	expected := []string{"cert-manager.io", "datarobot.com", "monitoring.coreos.com"}
	assert.Equal(t, expected, result.CRGroups)
}

func TestCRGroups_CoreSkipped(t *testing.T) {
	resources := []Resource{
		{Kind: "ConfigMap", APIVersion: "v1", Name: "cm", Namespace: "ns"},
		{Kind: "Secret", APIVersion: "v1", Name: "sec", Namespace: "ns"},
	}
	result := Classify(resources, nil)
	assert.Empty(t, result.CRGroups)
}

func TestCRGroups_K8sIoSkipped(t *testing.T) {
	resources := []Resource{
		{Kind: "NetworkPolicy", APIVersion: "networking.k8s.io/v1", Name: "np", Namespace: "ns"},
		{Kind: "SomeMetric", APIVersion: "metrics.k8s.io/v1beta1", Name: "m", Namespace: "ns"},
	}
	result := Classify(resources, nil)
	assert.Empty(t, result.CRGroups)
}

func TestCRGroups_AdminPartitionNotIncluded(t *testing.T) {
	// ClusterRole in rbac.authorization.k8s.io → Admin → must NOT appear in CRGroups.
	// Also a cluster-scoped CR via extraAdminKinds.
	resources := []Resource{
		{Kind: "ClusterRole", APIVersion: "rbac.authorization.k8s.io/v1", Name: "cr"},
		// A hypothetical cluster-scoped CR forced to Admin via extraAdminKinds
		{Kind: "MyClusterThing", APIVersion: "mycompany.io/v1", Name: "t"},
		// A namespaced resource in a third-party group → App → should appear
		{Kind: "Certificate", APIVersion: "cert-manager.io/v1", Name: "cert", Namespace: "ns"},
	}
	result := Classify(resources, []string{"MyClusterThing"})

	// rbac.authorization.k8s.io → .k8s.io → skipped even if it were App
	// mycompany.io → Admin → must NOT appear
	// cert-manager.io → App → must appear
	assert.Equal(t, []string{"cert-manager.io"}, result.CRGroups)
}

func TestCRGroups_CiliumIncluded(t *testing.T) {
	resources := []Resource{
		{Kind: "CiliumNetworkPolicy", APIVersion: "cilium.io/v2", Name: "cnp", Namespace: "ns"},
	}
	result := Classify(resources, nil)
	assert.Equal(t, []string{"cilium.io"}, result.CRGroups)
}

// F2(b): well-known CRD-backed .k8s.io groups in static exception set → included.
func TestCRGroups_GatewayNetworkingK8sIO_Included(t *testing.T) {
	resources := []Resource{
		// HTTPRoute in gateway.networking.k8s.io — CRD-backed, must be included despite suffix.
		{Kind: "HTTPRoute", APIVersion: "gateway.networking.k8s.io/v1", Name: "route", Namespace: "ns"},
		// Lease in coordination.k8s.io — built-in, must still be excluded.
		{Kind: "Lease", APIVersion: "coordination.k8s.io/v1", Name: "lease", Namespace: "ns"},
	}
	result := Classify(resources, nil)
	assert.Equal(t, []string{"gateway.networking.k8s.io"}, result.CRGroups,
		"gateway.networking.k8s.io included (crdBackedK8sIOGroups); coordination.k8s.io excluded (built-in)")
}

// F2(a): a group ending in .k8s.io with a chart-shipped CRD is always third-party.
func TestCRGroups_ChartShippedCRD_K8sIOGroup_Included(t *testing.T) {
	crdYAML := `apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: fancythings.custom.k8s.io
spec:
  group: custom.k8s.io
  names:
    kind: FancyThing
    plural: fancythings
  scope: Namespaced`

	resources := []Resource{
		{Kind: "CustomResourceDefinition", APIVersion: "apiextensions.k8s.io/v1", Name: "fancythings.custom.k8s.io", RawYAML: crdYAML},
		{Kind: "FancyThing", APIVersion: "custom.k8s.io/v1", Name: "ft", Namespace: "ns"},
	}
	result := Classify(resources, nil)
	assert.Equal(t, []string{"custom.k8s.io"}, result.CRGroups,
		"chart-shipped CRD with .k8s.io group overrides suffix heuristic → included")
}

// ---------------------------------------------------------------------------
// CRDResources tests
// ---------------------------------------------------------------------------

func TestCRDResources_TwoCRDsSameGroup(t *testing.T) {
	resources := []Resource{
		{
			Kind:       "CustomResourceDefinition",
			APIVersion: "apiextensions.k8s.io/v1",
			Name:       "widgets.example.com",
			RawYAML: `apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: widgets.example.com
spec:
  group: example.com
  names:
    plural: widgets
    kind: Widget
  scope: Namespaced`,
		},
		{
			Kind:       "CustomResourceDefinition",
			APIVersion: "apiextensions.k8s.io/v1",
			Name:       "gadgets.example.com",
			RawYAML: `apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: gadgets.example.com
spec:
  group: example.com
  names:
    plural: gadgets
    kind: Gadget
  scope: Namespaced`,
		},
	}

	got, warnings := CRDResources(resources)
	assert.Empty(t, warnings)
	assert.Equal(t, map[string][]string{
		"example.com": {"gadgets", "widgets"},
	}, got)
}

func TestCRDResources_MultipleGroups(t *testing.T) {
	resources := []Resource{
		{
			Kind:       "CustomResourceDefinition",
			APIVersion: "apiextensions.k8s.io/v1",
			Name:       "foos.alpha.io",
			RawYAML: `apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: foos.alpha.io
spec:
  group: alpha.io
  names:
    plural: foos
    kind: Foo
  scope: Cluster`,
		},
		{
			Kind:       "CustomResourceDefinition",
			APIVersion: "apiextensions.k8s.io/v1",
			Name:       "bars.beta.io",
			RawYAML: `apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: bars.beta.io
spec:
  group: beta.io
  names:
    plural: bars
    kind: Bar
  scope: Namespaced`,
		},
	}

	// foos.alpha.io is Cluster-scoped → excluded; bars.beta.io is Namespaced → included.
	got, warnings := CRDResources(resources)
	assert.Empty(t, warnings)
	assert.Equal(t, map[string][]string{
		"beta.io": {"bars"},
	}, got)
}

func TestCRDResources_NonCRDIgnored(t *testing.T) {
	resources := []Resource{
		{Kind: "Deployment", APIVersion: "apps/v1", Name: "app", Namespace: "ns",
			RawYAML: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: app
  namespace: ns`},
		{
			Kind:       "CustomResourceDefinition",
			APIVersion: "apiextensions.k8s.io/v1",
			Name:       "things.example.com",
			RawYAML: `apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: things.example.com
spec:
  group: example.com
  names:
    plural: things
    kind: Thing
  scope: Namespaced`,
		},
	}

	got, warnings := CRDResources(resources)
	assert.Empty(t, warnings)
	assert.Equal(t, map[string][]string{"example.com": {"things"}}, got)
}

func TestCRDResources_MissingPlural_Warning(t *testing.T) {
	resources := []Resource{
		{
			Kind:       "CustomResourceDefinition",
			APIVersion: "apiextensions.k8s.io/v1",
			Name:       "broken.example.com",
			RawYAML: `apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: broken.example.com
spec:
  group: example.com
  names:
    kind: Broken
  scope: Namespaced`,
		},
	}

	got, warnings := CRDResources(resources)
	require.Len(t, warnings, 1)
	assert.Contains(t, warnings[0], "broken.example.com")
	assert.Contains(t, warnings[0], "spec.names.plural")
	assert.Empty(t, got)
}

func TestCRDResources_MissingGroup_Warning(t *testing.T) {
	resources := []Resource{
		{
			Kind:       "CustomResourceDefinition",
			APIVersion: "apiextensions.k8s.io/v1",
			Name:       "nogroup.example.com",
			RawYAML: `apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: nogroup.example.com
spec:
  names:
    plural: nogroups
    kind: NoGroup
  scope: Namespaced`,
		},
	}

	got, warnings := CRDResources(resources)
	require.Len(t, warnings, 1)
	assert.Contains(t, warnings[0], "nogroup.example.com")
	assert.Contains(t, warnings[0], "spec.group")
	assert.Empty(t, got)
}

func TestCRDResources_Empty(t *testing.T) {
	got, warnings := CRDResources([]Resource{})
	assert.Empty(t, warnings)
	assert.Empty(t, got)
}

func TestCRDResources_DedupPlurals(t *testing.T) {
	// Same group+plural twice → deduplicated.
	yaml1 := `apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: widgets.example.com
spec:
  group: example.com
  names:
    plural: widgets
    kind: Widget
  scope: Namespaced`

	resources := []Resource{
		{Kind: "CustomResourceDefinition", APIVersion: "apiextensions.k8s.io/v1", Name: "widgets.example.com", RawYAML: yaml1},
		{Kind: "CustomResourceDefinition", APIVersion: "apiextensions.k8s.io/v1", Name: "widgets.example.com", RawYAML: yaml1},
	}

	got, warnings := CRDResources(resources)
	assert.Empty(t, warnings)
	assert.Equal(t, map[string][]string{"example.com": {"widgets"}}, got)
}

func TestCRDResources_ClusterScopedExcluded(t *testing.T) {
	// Cluster-scoped CRDs must be excluded; Namespaced included; missing scope = Namespaced.
	resources := []Resource{
		{
			Kind:       "CustomResourceDefinition",
			APIVersion: "apiextensions.k8s.io/v1",
			Name:       "clusters.infra.io",
			RawYAML: `apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: clusters.infra.io
spec:
  group: infra.io
  names:
    plural: clusters
    kind: Cluster
  scope: Cluster`,
		},
		{
			Kind:       "CustomResourceDefinition",
			APIVersion: "apiextensions.k8s.io/v1",
			Name:       "apps.infra.io",
			RawYAML: `apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: apps.infra.io
spec:
  group: infra.io
  names:
    plural: apps
    kind: App
  scope: Namespaced`,
		},
		{
			Kind:       "CustomResourceDefinition",
			APIVersion: "apiextensions.k8s.io/v1",
			Name:       "things.other.io",
			RawYAML: `apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: things.other.io
spec:
  group: other.io
  names:
    plural: things
    kind: Thing`,
		},
	}

	got, warnings := CRDResources(resources)
	assert.Empty(t, warnings)
	// clusters.infra.io (Cluster) excluded; apps.infra.io (Namespaced) included;
	// things.other.io (missing scope = Namespaced) included.
	assert.Equal(t, map[string][]string{
		"infra.io": {"apps"},
		"other.io": {"things"},
	}, got)
}

func TestCRDResources_MalformedCRD_Warning(t *testing.T) {
	// Malformed CRD produces warning; other CRDs still processed.
	resources := []Resource{
		{
			Kind:       "CustomResourceDefinition",
			APIVersion: "apiextensions.k8s.io/v1",
			Name:       "bad.example.com",
			RawYAML: `apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: bad.example.com
spec:
  group: example.com
  names:
    kind: Bad
  scope:
    - invalid
    - list`,
		},
		{
			Kind:       "CustomResourceDefinition",
			APIVersion: "apiextensions.k8s.io/v1",
			Name:       "goods.example.com",
			RawYAML: `apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: goods.example.com
spec:
  group: example.com
  names:
    plural: goods
    kind: Good
  scope: Namespaced`,
		},
	}

	got, warnings := CRDResources(resources)
	require.Len(t, warnings, 1)
	assert.Contains(t, warnings[0], "bad.example.com")
	assert.Equal(t, map[string][]string{"example.com": {"goods"}}, got)
}
