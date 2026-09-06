package manifest

import (
	"fmt"
	"sort"

	sigsyaml "sigs.k8s.io/yaml"
)

// minimalCRDFull parses the fields needed by CRDResources.
type minimalCRDFull struct {
	Metadata struct {
		Name string `json:"name"`
	} `json:"metadata"`
	Spec struct {
		Group string `json:"group"`
		Names struct {
			Plural string `json:"plural"`
		} `json:"names"`
		Scope string `json:"scope"`
	} `json:"spec"`
}

// CRDResources maps a CRD-provided API group to the sorted list of resource
// plurals (spec.names.plural) for Namespaced CustomResourceDefinitions only.
// Non-CRD kinds are ignored. CRDs with spec.scope == "Cluster" are excluded;
// missing or empty scope is treated as Namespaced (Kubernetes default).
// Per-CRD parse errors or missing required fields (spec.group,
// spec.names.plural) are returned as warnings; the offending CRD is skipped.
// The error return is reserved for future use and is always nil.
func CRDResources(resources []Resource) (map[string][]string, []string) {
	// group -> set of plurals
	groupPlurals := make(map[string]map[string]bool)
	var warnings []string

	for _, r := range resources {
		if r.Kind != "CustomResourceDefinition" {
			continue
		}

		var crd minimalCRDFull
		if err := sigsyaml.Unmarshal([]byte(r.RawYAML), &crd); err != nil {
			name := r.Name
			if name == "" {
				name = "<unknown>"
			}
			warnings = append(warnings, fmt.Sprintf("CRDResources: parse CRD %q: %v", name, err))
			continue
		}

		name := crd.Metadata.Name
		if name == "" {
			name = r.Name
		}
		if name == "" {
			name = "<unknown>"
		}

		if crd.Spec.Group == "" {
			warnings = append(warnings, fmt.Sprintf("CRDResources: CRD %q missing spec.group", name))
			continue
		}
		if crd.Spec.Names.Plural == "" {
			warnings = append(warnings, fmt.Sprintf("CRDResources: CRD %q missing spec.names.plural", name))
			continue
		}

		// Exclude cluster-scoped CRDs — aggregate ClusterRole must not grant
		// verbs on cluster-scoped resources to namespace-bound subjects.
		// Missing/empty scope treated as Namespaced (k8s default).
		if crd.Spec.Scope == "Cluster" {
			continue
		}

		if groupPlurals[crd.Spec.Group] == nil {
			groupPlurals[crd.Spec.Group] = make(map[string]bool)
		}
		groupPlurals[crd.Spec.Group][crd.Spec.Names.Plural] = true
	}

	if len(groupPlurals) == 0 {
		return map[string][]string{}, warnings
	}

	result := make(map[string][]string, len(groupPlurals))
	for g, pluralSet := range groupPlurals {
		plurals := make([]string, 0, len(pluralSet))
		for p := range pluralSet {
			plurals = append(plurals, p)
		}
		sort.Strings(plurals)
		result[g] = plurals
	}
	return result, warnings
}
