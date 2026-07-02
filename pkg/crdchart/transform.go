package crdchart

import (
	"bytes"
	"fmt"
	"strings"

	yaml "gopkg.in/yaml.v3"
)

// AddKeepAnnotation sets helm.sh/resource-policy: keep on the resource,
// preserving any existing metadata/annotations, and returns the updated
// Resource with a re-marshaled RawYAML.
//
// It uses a yaml.Node round-trip rather than a map[string]interface{}
// round-trip so that sibling key order and 2-space indentation are preserved
// (a plain map marshal sorts keys alphabetically and re-indents to 4 spaces),
// and scalar fidelity (large integers, quoted boolean-like strings) is kept.
func AddKeepAnnotation(r Resource) (Resource, error) {
	const annKey = "helm.sh/resource-policy"
	const annVal = "keep"

	var docNode yaml.Node
	if err := yaml.Unmarshal([]byte(r.RawYAML), &docNode); err != nil {
		return r, err
	}
	if docNode.Kind != yaml.DocumentNode || len(docNode.Content) == 0 {
		return r, fmt.Errorf("AddKeepAnnotation: unexpected document structure")
	}
	root := docNode.Content[0]

	metaNode := findOrCreateMapping(root, "metadata")
	annoNode := findOrCreateMapping(metaNode, "annotations")

	// Set or overwrite the annotation key.
	set := false
	for i := 0; i+1 < len(annoNode.Content); i += 2 {
		if annoNode.Content[i].Value == annKey {
			annoNode.Content[i+1].Value = annVal
			annoNode.Content[i+1].Tag = "!!str"
			set = true
			break
		}
	}
	if !set {
		annoNode.Content = append(annoNode.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: annKey},
			&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: annVal},
		)
	}

	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(&docNode); err != nil {
		return r, fmt.Errorf("AddKeepAnnotation marshal: %w", err)
	}
	_ = enc.Close()
	r.RawYAML = strings.TrimRight(buf.String(), "\n")
	return r, nil
}

// StripKeepAnnotation removes the helm.sh/resource-policy annotation from the
// resource, dropping the annotations map entirely if it becomes empty as a
// result, and returns the updated Resource with a re-marshaled RawYAML. It is
// a no-op (r unchanged, nil error) when metadata/annotations/the key are
// absent.
//
// Like AddKeepAnnotation, it uses a yaml.Node round-trip rather than a
// map[string]interface{} round-trip so that sibling key order and 2-space
// indentation are preserved (a plain map marshal sorts keys alphabetically
// and re-indents to 4 spaces), and scalar fidelity is kept.
func StripKeepAnnotation(r Resource) (Resource, error) {
	const annKey = "helm.sh/resource-policy"

	var docNode yaml.Node
	if err := yaml.Unmarshal([]byte(r.RawYAML), &docNode); err != nil {
		return r, err
	}
	if docNode.Kind != yaml.DocumentNode || len(docNode.Content) == 0 {
		return r, fmt.Errorf("StripKeepAnnotation: unexpected document structure")
	}
	root := docNode.Content[0]

	metaNode := findMapping(root, "metadata")
	if metaNode == nil {
		return r, nil
	}
	annoNode := findMapping(metaNode, "annotations")
	if annoNode == nil {
		return r, nil
	}

	idx := -1
	for i := 0; i+1 < len(annoNode.Content); i += 2 {
		if annoNode.Content[i].Value == annKey {
			idx = i
			break
		}
	}
	if idx == -1 {
		return r, nil
	}
	annoNode.Content = append(annoNode.Content[:idx], annoNode.Content[idx+2:]...)

	if len(annoNode.Content) == 0 {
		removeKey(metaNode, "annotations")
	}

	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(&docNode); err != nil {
		return r, fmt.Errorf("StripKeepAnnotation marshal: %w", err)
	}
	_ = enc.Close()
	r.RawYAML = strings.TrimRight(buf.String(), "\n")
	return r, nil
}

// findMapping returns the value node for key within the parent mapping, or
// nil if key is absent or its value is not itself a mapping.
func findMapping(parent *yaml.Node, key string) *yaml.Node {
	for i := 0; i+1 < len(parent.Content); i += 2 {
		if parent.Content[i].Value == key {
			if parent.Content[i+1].Kind != yaml.MappingNode {
				return nil
			}
			return parent.Content[i+1]
		}
	}
	return nil
}

// removeKey deletes key (and its value) from parent's mapping content, if present.
func removeKey(parent *yaml.Node, key string) {
	for i := 0; i+1 < len(parent.Content); i += 2 {
		if parent.Content[i].Value == key {
			parent.Content = append(parent.Content[:i], parent.Content[i+2:]...)
			return
		}
	}
}

// findOrCreateMapping returns the value node for key within the parent mapping,
// creating an empty mapping (and appending the key) if it is absent or is not
// itself a mapping.
func findOrCreateMapping(parent *yaml.Node, key string) *yaml.Node {
	for i := 0; i+1 < len(parent.Content); i += 2 {
		if parent.Content[i].Value == key {
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

// EscapeBraces neutralizes Go-template delimiters that appear inside CRD
// bodies (CEL rules, descriptions) so Helm renders them back literally
// instead of trying to evaluate them.
func EscapeBraces(raw string) string {
	const lo = "\x00LBRACE\x00"
	const ro = "\x00RBRACE\x00"
	s := strings.ReplaceAll(raw, "{{", lo)
	s = strings.ReplaceAll(s, "}}", ro)
	s = strings.ReplaceAll(s, lo, `{{ "{{" }}`)
	s = strings.ReplaceAll(s, ro, `{{ "}}" }}`)
	return s
}
