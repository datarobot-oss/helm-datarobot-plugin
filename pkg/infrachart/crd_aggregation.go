package infrachart

import (
	"fmt"
	"sort"

	"github.com/datarobot-oss/helm-datarobot-plugin/pkg/manifest"
)

// BuildCRDAggregation generates an aggregate-to-admin/edit ClusterRole granting
// full access to the CRD-provided resources the admin chart ships. This is the
// industry-standard pattern (cert-manager-edit) DR's operator charts don't ship
// yet; generating it from the extracted CRDs closes the gap for every
// namespace-admin subject in the cluster, not only the pipeline SA.
// crdResources: group -> sorted resource plurals. Returns nil, nil when empty.
func BuildCRDAggregation(crdResources map[string][]string, releaseName string) ([]manifest.Resource, error) {
	if releaseName == "" {
		return nil, fmt.Errorf("BuildCRDAggregation: releaseName must not be empty")
	}
	if len(crdResources) == 0 {
		return nil, nil
	}

	// Sort groups for deterministic rule order.
	groups := make([]string, 0, len(crdResources))
	for g := range crdResources {
		groups = append(groups, g)
	}
	sort.Strings(groups)

	rules := make([]map[string]interface{}, 0, len(groups))
	for _, g := range groups {
		rules = append(rules, map[string]interface{}{
			"apiGroups": []string{g},
			"resources": crdResources[g],
			"verbs":     []string{"*"},
		})
	}

	name := releaseName + "-crd-edit-aggregate"
	r, err := marshalResource("ClusterRole", "rbac.authorization.k8s.io/v1", name, "", map[string]interface{}{
		"apiVersion": "rbac.authorization.k8s.io/v1",
		"kind":       "ClusterRole",
		"metadata": map[string]interface{}{
			"name": name,
			"labels": map[string]interface{}{
				"rbac.authorization.k8s.io/aggregate-to-admin": "true",
				"rbac.authorization.k8s.io/aggregate-to-edit":  "true",
			},
		},
		"rules": rules,
	})
	if err != nil {
		return nil, fmt.Errorf("ClusterRole crd-edit-aggregate: %w", err)
	}
	return []manifest.Resource{r}, nil
}
