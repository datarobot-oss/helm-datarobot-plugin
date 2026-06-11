// Package adminchart builds the admin chart from cluster-scoped resources.
//
// This file handles bootstrap RBAC for the pipeline ServiceAccount.
// Two design constraints it addresses:
//  1. Charts use `lookup` on cluster-scoped kinds at render time, so the SA
//     needs a ClusterRole granting get/list/watch on those kinds.
//  2. RBAC escalation prevention: the SA must hold the union of all app-chart
//     Role rules before Helm will allow it to install/upgrade those app charts
//     (a controller/principal cannot grant permissions it doesn't itself hold).
package adminchart

import (
	"fmt"
	"strings"

	sigsyaml "sigs.k8s.io/yaml"

	"github.com/datarobot-oss/helm-datarobot-plugin/pkg/manifest"
)

// DefaultClusterReadKinds is the maintained default list for ClusterReadRules.
// Each entry is "resource.group" or "resource" (core group, no dot).
var DefaultClusterReadKinds = []string{
	"storageclasses.storage.k8s.io",
	"namespaces",
	"customresourcedefinitions.apiextensions.k8s.io",
}

// PipelineRBACOptions parameterises BuildPipelineRBAC.
type PipelineRBACOptions struct {
	SAName      string
	Namespace   string
	ReleaseName string
	// ClusterReadRules are the policy rules for the cluster-read ClusterRole
	// (address finding 1: lookup on cluster-scoped kinds at render time).
	ClusterReadRules []map[string]interface{}
	// UnionRules is the union of all app-chart Role rules.
	// Empty means skip the union ClusterRole + RoleBinding (finding 2 not needed).
	UnionRules []map[string]interface{}
}

// ClusterReadRules converts "resource.group" / "resource" (core group) kind specs
// into get/list/watch PolicyRule maps suitable for a ClusterRole.rules field.
//
// Parsing: split on FIRST "."; left = resource (plural), rest = apiGroup
// ("" if no dot present).
//
// Each entry is TrimSpace'd; blank entries are skipped. An entry whose resource
// part (before the first ".") is empty (e.g. ".somegroup") is an error.
//
// Example input:
//
//	["storageclasses.storage.k8s.io", "namespaces", "customresourcedefinitions.apiextensions.k8s.io"]
func ClusterReadRules(kinds []string) ([]map[string]interface{}, error) {
	rules := make([]map[string]interface{}, 0, len(kinds))
	for _, k := range kinds {
		k = strings.TrimSpace(k)
		if k == "" {
			continue
		}
		var resource, apiGroup string
		if idx := strings.Index(k, "."); idx >= 0 {
			resource = k[:idx]
			apiGroup = k[idx+1:]
		} else {
			resource = k
			apiGroup = ""
		}
		if resource == "" {
			return nil, fmt.Errorf("ClusterReadRules: invalid entry %q: resource part (before first \".\") must not be empty", k)
		}
		rules = append(rules, map[string]interface{}{
			"apiGroups": []string{apiGroup},
			"resources": []string{resource},
			"verbs":     []string{"get", "list", "watch"},
		})
	}
	return rules, nil
}

// BuildPipelineRBAC renders the bootstrap RBAC manifests for the pipeline SA.
// Emits (in order):
//  1. ServiceAccount <SAName> in Namespace
//  2. RoleBinding <ReleaseName>-pipeline-admin in Namespace → ClusterRole admin
//  3. ClusterRole <ReleaseName>-pipeline-cluster-read (ClusterReadRules)
//  4. ClusterRoleBinding <ReleaseName>-pipeline-cluster-read → SA
//  5. (if UnionRules non-empty) ClusterRole <ReleaseName>-pipeline-role-union
//  6. (if UnionRules non-empty) RoleBinding <ReleaseName>-pipeline-role-union in Namespace → union CR
//
// Note on (6): deliberately NOT an aggregate-to-admin label to avoid cluster-wide side effects.
func BuildPipelineRBAC(opts PipelineRBACOptions) ([]manifest.Resource, error) {
	if opts.SAName == "" {
		return nil, fmt.Errorf("PipelineRBACOptions.SAName must not be empty")
	}
	if opts.Namespace == "" {
		return nil, fmt.Errorf("PipelineRBACOptions.Namespace must not be empty")
	}
	if opts.ReleaseName == "" {
		return nil, fmt.Errorf("PipelineRBACOptions.ReleaseName must not be empty")
	}

	var resources []manifest.Resource

	// 1. ServiceAccount
	sa, err := marshalResource("ServiceAccount", "v1", opts.SAName, opts.Namespace, map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "ServiceAccount",
		"metadata": map[string]interface{}{
			"name":      opts.SAName,
			"namespace": opts.Namespace,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("ServiceAccount: %w", err)
	}
	resources = append(resources, sa)

	// 2. RoleBinding → ClusterRole admin
	rb1Name := opts.ReleaseName + "-pipeline-admin"
	rb1, err := marshalResource("RoleBinding", "rbac.authorization.k8s.io/v1", rb1Name, opts.Namespace, map[string]interface{}{
		"apiVersion": "rbac.authorization.k8s.io/v1",
		"kind":       "RoleBinding",
		"metadata": map[string]interface{}{
			"name":      rb1Name,
			"namespace": opts.Namespace,
		},
		"roleRef": map[string]interface{}{
			"apiGroup": "rbac.authorization.k8s.io",
			"kind":     "ClusterRole",
			"name":     "admin",
		},
		"subjects": []map[string]interface{}{
			{
				"kind":      "ServiceAccount",
				"name":      opts.SAName,
				"namespace": opts.Namespace,
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("RoleBinding pipeline-admin: %w", err)
	}
	resources = append(resources, rb1)

	// 3. ClusterRole cluster-read
	crReadName := opts.ReleaseName + "-pipeline-cluster-read"
	crRead, err := marshalResource("ClusterRole", "rbac.authorization.k8s.io/v1", crReadName, "", map[string]interface{}{
		"apiVersion": "rbac.authorization.k8s.io/v1",
		"kind":       "ClusterRole",
		"metadata": map[string]interface{}{
			"name": crReadName,
		},
		"rules": opts.ClusterReadRules,
	})
	if err != nil {
		return nil, fmt.Errorf("ClusterRole pipeline-cluster-read: %w", err)
	}
	resources = append(resources, crRead)

	// 4. ClusterRoleBinding cluster-read → SA
	crbReadName := opts.ReleaseName + "-pipeline-cluster-read"
	crbRead, err := marshalResource("ClusterRoleBinding", "rbac.authorization.k8s.io/v1", crbReadName, "", map[string]interface{}{
		"apiVersion": "rbac.authorization.k8s.io/v1",
		"kind":       "ClusterRoleBinding",
		"metadata": map[string]interface{}{
			"name": crbReadName,
		},
		"roleRef": map[string]interface{}{
			"apiGroup": "rbac.authorization.k8s.io",
			"kind":     "ClusterRole",
			"name":     crReadName,
		},
		"subjects": []map[string]interface{}{
			{
				"kind":      "ServiceAccount",
				"name":      opts.SAName,
				"namespace": opts.Namespace,
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("ClusterRoleBinding pipeline-cluster-read: %w", err)
	}
	resources = append(resources, crbRead)

	if len(opts.UnionRules) > 0 {
		// 5. ClusterRole role-union
		crUnionName := opts.ReleaseName + "-pipeline-role-union"
		crUnion, err := marshalResource("ClusterRole", "rbac.authorization.k8s.io/v1", crUnionName, "", map[string]interface{}{
			"apiVersion": "rbac.authorization.k8s.io/v1",
			"kind":       "ClusterRole",
			"metadata": map[string]interface{}{
				"name": crUnionName,
			},
			"rules": opts.UnionRules,
		})
		if err != nil {
			return nil, fmt.Errorf("ClusterRole pipeline-role-union: %w", err)
		}
		resources = append(resources, crUnion)

		// 6. RoleBinding role-union → SA (namespace-scoped, no aggregation label)
		rb2Name := opts.ReleaseName + "-pipeline-role-union"
		rb2, err := marshalResource("RoleBinding", "rbac.authorization.k8s.io/v1", rb2Name, opts.Namespace, map[string]interface{}{
			"apiVersion": "rbac.authorization.k8s.io/v1",
			"kind":       "RoleBinding",
			"metadata": map[string]interface{}{
				"name":      rb2Name,
				"namespace": opts.Namespace,
			},
			"roleRef": map[string]interface{}{
				"apiGroup": "rbac.authorization.k8s.io",
				"kind":     "ClusterRole",
				"name":     crUnionName,
			},
			"subjects": []map[string]interface{}{
				{
					"kind":      "ServiceAccount",
					"name":      opts.SAName,
					"namespace": opts.Namespace,
				},
			},
		})
		if err != nil {
			return nil, fmt.Errorf("RoleBinding pipeline-role-union: %w", err)
		}
		resources = append(resources, rb2)
	}

	return resources, nil
}

// marshalResource converts a map to a manifest.Resource via sigs.k8s.io/yaml.
func marshalResource(kind, apiVersion, name, namespace string, obj map[string]interface{}) (manifest.Resource, error) {
	raw, err := sigsyaml.Marshal(obj)
	if err != nil {
		return manifest.Resource{}, fmt.Errorf("marshal %s/%s: %w", kind, name, err)
	}
	return manifest.Resource{
		Kind:       kind,
		APIVersion: apiVersion,
		Name:       name,
		Namespace:  namespace,
		RawYAML:    strings.TrimSpace(string(raw)),
	}, nil
}
