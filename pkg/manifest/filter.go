package manifest

import (
	"fmt"
	"sort"
	"strings"

	"sigs.k8s.io/yaml"
)

// Resource represents a single Kubernetes resource extracted from a Helm manifest.
type Resource struct {
	Kind      string
	Name      string
	Namespace string
	RawYAML   string
}

// minimalResource is used for parsing only the fields we need.
type minimalResource struct {
	Kind     string `json:"kind"`
	Metadata struct {
		Name      string `json:"name"`
		Namespace string `json:"namespace"`
	} `json:"metadata"`
}

// ParseManifests splits multi-document YAML and returns parsed resources.
// Empty documents and comment-only documents are skipped.
// Documents with empty kind are skipped.
func ParseManifests(manifest string) ([]Resource, error) {
	docs := strings.Split(manifest, "---")
	var resources []Resource

	for _, doc := range docs {
		trimmed := strings.TrimSpace(doc)
		if trimmed == "" {
			continue
		}

		// Skip comment-only documents
		onlyComments := true
		for _, line := range strings.Split(trimmed, "\n") {
			l := strings.TrimSpace(line)
			if l != "" && !strings.HasPrefix(l, "#") {
				onlyComments = false
				break
			}
		}
		if onlyComments {
			continue
		}

		var r minimalResource
		if err := yaml.Unmarshal([]byte(trimmed), &r); err != nil {
			return nil, fmt.Errorf("parse manifest: %w", err)
		}

		if r.Kind == "" {
			continue
		}

		resources = append(resources, Resource{
			Kind:      r.Kind,
			Name:      r.Metadata.Name,
			Namespace: r.Metadata.Namespace,
			RawYAML:   trimmed,
		})
	}

	return resources, nil
}

// FilterClusterScoped separates resources into cluster-scoped (no namespace) and namespaced.
func FilterClusterScoped(resources []Resource) (clusterScoped, namespaced []Resource) {
	for _, r := range resources {
		if r.Namespace == "" {
			clusterScoped = append(clusterScoped, r)
		} else {
			namespaced = append(namespaced, r)
		}
	}
	return
}

// GroupByKind groups resources by their Kind field.
func GroupByKind(resources []Resource) map[string][]Resource {
	result := make(map[string][]Resource)
	for _, r := range resources {
		result[r.Kind] = append(result[r.Kind], r)
	}
	return result
}

// Summary returns a human-readable sorted summary of resource counts by kind.
// Example: "ClusterRole: 2, CustomResourceDefinition: 3"
func Summary(resources []Resource) string {
	groups := GroupByKind(resources)

	kinds := make([]string, 0, len(groups))
	for k := range groups {
		kinds = append(kinds, k)
	}
	sort.Strings(kinds)

	parts := make([]string, 0, len(kinds))
	for _, k := range kinds {
		parts = append(parts, fmt.Sprintf("%s: %d", k, len(groups[k])))
	}
	return strings.Join(parts, ", ")
}
