package manifest

import (
	"bytes"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// StripKeepAnnotation removes the helm.sh/resource-policy annotation from the
// resource, dropping the annotations map entirely if it becomes empty as a
// result, and returns the updated Resource with a re-marshaled RawYAML. It is
// a no-op (r unchanged, nil error) when metadata/annotations/the key are
// absent.
//
// Ported from pkg/crdchart's StripKeepAnnotation (Phase 1, ticket CRD-001):
// some source charts bake helm.sh/resource-policy: keep into their CRDs at
// template time regardless of any --keep-crds flag on this tool. Without this
// strip step, --keep-crds=false would fail to actually drop the annotation
// for those charts.
//
// Uses a yaml.Node round-trip rather than a map[string]interface{} round-trip
// so that sibling key order and 2-space indentation are preserved (a plain
// map marshal sorts keys alphabetically and re-indents to 4 spaces), and
// scalar fidelity is kept.
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

	metaNode := mappingValue(root, "metadata")
	if metaNode == nil {
		return r, nil
	}
	annoNode := mappingValue(metaNode, "annotations")
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

// removeKey deletes key (and its value) from parent's mapping content, if present.
func removeKey(parent *yaml.Node, key string) {
	for i := 0; i+1 < len(parent.Content); i += 2 {
		if parent.Content[i].Value == key {
			parent.Content = append(parent.Content[:i], parent.Content[i+2:]...)
			return
		}
	}
}
