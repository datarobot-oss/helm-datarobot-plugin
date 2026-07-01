package crdchart

import (
	"regexp"
	"strings"

	yaml "gopkg.in/yaml.v3"
)

// docSepRe matches a YAML document separator: a line that is exactly "---"
// with an optional trailing comment and/or trailing whitespace. Anchoring to
// the start of a line (multiline mode) avoids false splits on content that
// merely contains "---" — indented block-scalar text (CRD descriptions, CEL
// messages) or PEM blocks like "-----BEGIN CERTIFICATE-----". It also matches
// separators that carry a trailing comment ("--- # Source: ...") or trailing
// spaces, which a plain strings.Split(manifest, "\n---\n") would miss and
// thereby glue two documents together.
var docSepRe = regexp.MustCompile(`(?m)^---(\s+#.*)?[ \t]*$`)

// Resource is one rendered Kubernetes object plus its exact source YAML.
type Resource struct {
	Kind      string
	Name      string
	Namespace string
	RawYAML   string
}

type metaHeader struct {
	Kind     string `yaml:"kind"`
	Metadata struct {
		Name      string `yaml:"name"`
		Namespace string `yaml:"namespace"`
	} `yaml:"metadata"`
}

// ParseManifests splits a Helm multi-doc render into typed Resources.
// Documents that are empty, comment-only, or carry no kind are skipped.
func ParseManifests(manifest string) ([]Resource, error) {
	var out []Resource
	for _, doc := range docSepRe.Split(manifest, -1) {
		trimmed := strings.TrimSpace(doc)
		if trimmed == "" {
			continue
		}
		// Skip comment-only documents (every non-blank line starts with '#'),
		// which would otherwise unmarshal to an empty header with no kind.
		if isCommentOnly(trimmed) {
			continue
		}
		var h metaHeader
		if err := yaml.Unmarshal([]byte(trimmed), &h); err != nil {
			return nil, err
		}
		if h.Kind == "" {
			continue
		}
		out = append(out, Resource{
			Kind:      h.Kind,
			Name:      h.Metadata.Name,
			Namespace: h.Metadata.Namespace,
			RawYAML:   trimmed,
		})
	}
	return out, nil
}

// isCommentOnly reports whether every non-blank line in doc is a YAML comment.
func isCommentOnly(doc string) bool {
	for _, line := range strings.Split(doc, "\n") {
		l := strings.TrimSpace(line)
		if l != "" && !strings.HasPrefix(l, "#") {
			return false
		}
	}
	return true
}

// KeepMatching returns only the resources for which keep returns true.
// The predicate is the Phase 2 seam (infra-chart passes isClusterScoped).
func KeepMatching(resources []Resource, keep func(Resource) bool) []Resource {
	var out []Resource
	for _, r := range resources {
		if keep(r) {
			out = append(out, r)
		}
	}
	return out
}

// IsCRD reports whether r is a CustomResourceDefinition.
func IsCRD(r Resource) bool {
	return r.Kind == "CustomResourceDefinition"
}

// DedupeByName keeps the last occurrence of each metadata.name and reports
// which names collided so the caller can warn.
func DedupeByName(resources []Resource) (kept []Resource, collisions []string) {
	idx := make(map[string]int)
	seen := make(map[string]bool)
	for _, r := range resources {
		if pos, ok := idx[r.Name]; ok {
			kept[pos] = r // last wins
			if !seen[r.Name] {
				collisions = append(collisions, r.Name)
				seen[r.Name] = true
			}
			continue
		}
		idx[r.Name] = len(kept)
		kept = append(kept, r)
	}
	return kept, collisions
}
