package manifest

import (
	"bytes"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
	sigsyaml "sigs.k8s.io/yaml"
)

// docSepRe matches YAML document separator lines: "---" with optional trailing comment and whitespace.
// Note: "--- {inline: doc}" same-line YAML content is not supported as a separator.
var docSepRe = regexp.MustCompile(`(?m)^---(\s+#.*)?\s*$`)

// Resource represents a single Kubernetes resource extracted from a Helm manifest.
type Resource struct {
	Kind       string
	APIVersion string
	Name       string
	Namespace  string
	RawYAML    string
}

// minimalResource is used for parsing only the fields we need.
type minimalResource struct {
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
	Metadata   struct {
		Name      string `json:"name"`
		Namespace string `json:"namespace"`
	} `json:"metadata"`
}

// ParseManifests splits multi-document YAML and returns parsed resources.
// Empty documents and comment-only documents are skipped.
// Documents with empty kind are skipped.
// Splitting uses line-start "---" separators only, avoiding false splits on
// PEM blocks (-----BEGIN CERTIFICATE-----) or description strings containing "---".
func ParseManifests(manifest string) ([]Resource, error) {
	// Split on lines that are exactly "---" (optional trailing whitespace).
	// The regex matches the separator line itself; we split the text around those positions.
	// We include the trailing newline in the separator so docs don't start with a blank line.
	parts := docSepRe.Split(manifest, -1)

	var resources []Resource

	for _, doc := range parts {
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
		if err := sigsyaml.Unmarshal([]byte(trimmed), &r); err != nil {
			return nil, fmt.Errorf("parse manifest: %w", err)
		}

		if r.Kind == "" {
			continue
		}

		resources = append(resources, Resource{
			Kind:       r.Kind,
			APIVersion: r.APIVersion,
			Name:       r.Metadata.Name,
			Namespace:  r.Metadata.Namespace,
			RawYAML:    trimmed,
		})
	}

	return resources, nil
}

// ClassifyResult holds the output of Classify.
type ClassifyResult struct {
	Admin    []Resource
	App      []Resource
	Warnings []string
}

// crdInfo holds parsed CRD scope info used during Classify.
type crdInfo struct {
	group string
	kind  string
	scope string // "Cluster" or "Namespaced"
}

// minimalCRD is used to unmarshal only the CRD fields we need.
type minimalCRD struct {
	Spec struct {
		Group string `json:"group"`
		Names struct {
			Kind string `json:"kind"`
		} `json:"names"`
		Scope string `json:"scope"`
	} `json:"spec"`
}

// Classify splits resources into Admin (cluster-scoped + forced kinds) and App (namespaced).
// Precedence per resource:
//  1. Kind in extraAdminKinds (case-insensitive) -> Admin
//  2. Kind matches a CRD in the input set -> use CRD spec.scope (Cluster->Admin, Namespaced->App)
//  3. Kind in built-in static scope tables -> per table
//  4. Fallback: namespace-field heuristic + warning
func Classify(resources []Resource, extraAdminKinds []string) ClassifyResult {
	// Build extraAdminKinds set (lowercase for case-insensitive match).
	extraSet := make(map[string]bool, len(extraAdminKinds))
	for _, k := range extraAdminKinds {
		extraSet[strings.ToLower(k)] = true
	}

	// Build CRD lookup: key = "<group>/<kind>" (lowercased).
	crdLookup := make(map[string]crdInfo)

	var result ClassifyResult

	for _, r := range resources {
		if r.Kind != "CustomResourceDefinition" {
			continue
		}
		var crd minimalCRD
		if err := sigsyaml.Unmarshal([]byte(r.RawYAML), &crd); err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("failed to parse CRD %s: %s", r.Name, err))
			continue
		}
		key := strings.ToLower(crd.Spec.Group + "/" + crd.Spec.Names.Kind)
		crdLookup[key] = crdInfo{
			group: crd.Spec.Group,
			kind:  crd.Spec.Names.Kind,
			scope: crd.Spec.Scope,
		}
	}

	for _, r := range resources {
		dest := classifyOne(r, extraSet, crdLookup, &result.Warnings)
		if dest == "admin" {
			result.Admin = append(result.Admin, r)
		} else {
			result.App = append(result.App, r)
		}
	}

	return result
}

// classifyOne returns "admin" or "app" for a single resource.
func classifyOne(r Resource, extraSet map[string]bool, crdLookup map[string]crdInfo, warnings *[]string) string {
	// 1. extraAdminKinds override.
	if extraSet[strings.ToLower(r.Kind)] {
		return "admin"
	}

	// 2. CRD-based scope: parse group from apiVersion, look up in crdLookup.
	group := apiVersionGroup(r.APIVersion)
	if group != "" {
		key := strings.ToLower(group + "/" + r.Kind)
		if info, ok := crdLookup[key]; ok {
			if info.scope == "Cluster" {
				return "admin"
			}
			return "app"
		}
	}

	// 3. Static table lookup.
	if clusterScopedKinds[r.Kind] {
		return "admin"
	}
	if namespacedKinds[r.Kind] {
		return "app"
	}

	// 4. Fallback: namespace-field heuristic + warning.
	dest := "app"
	if r.Namespace == "" {
		dest = "admin"
	}
	av := r.APIVersion
	if av == "" {
		av = "unknown"
	}
	name := r.Name
	if r.Namespace != "" {
		name = r.Namespace + "/" + r.Name
	}
	*warnings = append(*warnings, fmt.Sprintf(
		"unknown kind %s (%s) %s: falling back to namespace-field heuristic, classified as %s",
		r.Kind, av, name, dest,
	))
	return dest
}

// apiVersionGroup extracts the group from an apiVersion string.
// "apps/v1" -> "apps", "v1" -> "", "apiextensions.k8s.io/v1" -> "apiextensions.k8s.io"
func apiVersionGroup(apiVersion string) string {
	parts := strings.SplitN(apiVersion, "/", 2)
	if len(parts) == 2 {
		return parts[0]
	}
	return ""
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

// WithAnnotation returns a copy of r with metadata.annotations[key]=value set in RawYAML.
// Uses gopkg.in/yaml.v3 Node round-trip to preserve scalar fidelity
// (large integers, unquoted boolean-like strings, etc.).
func (r Resource) WithAnnotation(key, value string) (Resource, error) {
	var docNode yaml.Node
	if err := yaml.Unmarshal([]byte(r.RawYAML), &docNode); err != nil {
		return r, fmt.Errorf("WithAnnotation unmarshal: %w", err)
	}
	if docNode.Kind != yaml.DocumentNode || len(docNode.Content) == 0 {
		return r, fmt.Errorf("WithAnnotation: unexpected document structure")
	}
	root := docNode.Content[0]

	metaNode := findOrCreateMapping(root, "metadata")
	annoNode := findOrCreateMapping(metaNode, "annotations")

	// Set or overwrite key.
	for i := 0; i < len(annoNode.Content)-1; i += 2 {
		if annoNode.Content[i].Value == key {
			annoNode.Content[i+1].Value = value
			annoNode.Content[i+1].Tag = "!!str"
			goto marshal
		}
	}
	// Key not found — append.
	annoNode.Content = append(annoNode.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key},
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: value},
	)

marshal:
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(&docNode); err != nil {
		return r, fmt.Errorf("WithAnnotation marshal: %w", err)
	}
	_ = enc.Close()
	newR := r
	newR.RawYAML = strings.TrimSpace(buf.String())
	return newR, nil
}

// StripHelmHookAnnotations returns a copy of r with all metadata.annotations keys
// having prefix "helm.sh/hook" removed.
// Fast path: if RawYAML doesn't contain "helm.sh/hook", returns r unchanged.
// If annotations becomes empty after removal, the annotations map is removed entirely.
// Uses gopkg.in/yaml.v3 Node round-trip to preserve scalar fidelity.
func (r Resource) StripHelmHookAnnotations() (Resource, error) {
	if !strings.Contains(r.RawYAML, "helm.sh/hook") {
		return r, nil
	}

	var docNode yaml.Node
	if err := yaml.Unmarshal([]byte(r.RawYAML), &docNode); err != nil {
		return r, fmt.Errorf("StripHelmHookAnnotations unmarshal: %w", err)
	}
	if docNode.Kind != yaml.DocumentNode || len(docNode.Content) == 0 {
		return r, fmt.Errorf("StripHelmHookAnnotations: unexpected document structure")
	}
	root := docNode.Content[0]

	metaNode := mappingValue(root, "metadata")
	if metaNode == nil {
		return r, nil
	}
	annoNode := mappingValue(metaNode, "annotations")
	if annoNode == nil {
		return r, nil
	}

	// Remove helm.sh/hook* key-value pairs.
	kept := make([]*yaml.Node, 0, len(annoNode.Content))
	for i := 0; i < len(annoNode.Content)-1; i += 2 {
		if !strings.HasPrefix(annoNode.Content[i].Value, "helm.sh/hook") {
			kept = append(kept, annoNode.Content[i], annoNode.Content[i+1])
		}
	}

	if len(kept) == 0 {
		// Remove annotations mapping from metadata.
		newContent := make([]*yaml.Node, 0, len(metaNode.Content))
		for i := 0; i < len(metaNode.Content)-1; i += 2 {
			if metaNode.Content[i].Value != "annotations" {
				newContent = append(newContent, metaNode.Content[i], metaNode.Content[i+1])
			}
		}
		metaNode.Content = newContent
	} else {
		annoNode.Content = kept
	}

	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(&docNode); err != nil {
		return r, fmt.Errorf("StripHelmHookAnnotations marshal: %w", err)
	}
	_ = enc.Close()
	newR := r
	newR.RawYAML = strings.TrimSpace(buf.String())
	return newR, nil
}

// findOrCreateMapping finds the value node for key in a MappingNode, creating it if absent.
func findOrCreateMapping(parent *yaml.Node, key string) *yaml.Node {
	for i := 0; i < len(parent.Content)-1; i += 2 {
		if parent.Content[i].Value == key {
			// Ensure value node is a mapping.
			if parent.Content[i+1].Kind != yaml.MappingNode {
				parent.Content[i+1] = &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
			}
			return parent.Content[i+1]
		}
	}
	valNode := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	parent.Content = append(parent.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key},
		valNode,
	)
	return valNode
}

// mappingValue returns the value node for key in a MappingNode, or nil if not found.
func mappingValue(parent *yaml.Node, key string) *yaml.Node {
	for i := 0; i < len(parent.Content)-1; i += 2 {
		if parent.Content[i].Value == key {
			return parent.Content[i+1]
		}
	}
	return nil
}
