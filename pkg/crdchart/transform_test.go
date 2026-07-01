package crdchart

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	yaml "gopkg.in/yaml.v3"
)

func TestAddKeepAnnotation(t *testing.T) {
	r := Resource{
		Kind: "CustomResourceDefinition",
		Name: "widgets.example.com",
		RawYAML: `apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: widgets.example.com
spec: {}`,
	}
	got, err := AddKeepAnnotation(r)
	assert.NoError(t, err)

	var m map[string]interface{}
	assert.NoError(t, yaml.Unmarshal([]byte(got.RawYAML), &m))
	meta := m["metadata"].(map[string]interface{})
	ann := meta["annotations"].(map[string]interface{})
	assert.Equal(t, "keep", ann["helm.sh/resource-policy"])
	// existing fields preserved
	assert.Equal(t, "widgets.example.com", meta["name"])
	assert.Equal(t, "CustomResourceDefinition", m["kind"])
}

func TestAddKeepAnnotationPreservesExistingAnnotations(t *testing.T) {
	r := Resource{RawYAML: `kind: CustomResourceDefinition
metadata:
  name: x
  annotations:
    foo: bar`}
	got, err := AddKeepAnnotation(r)
	assert.NoError(t, err)
	var m map[string]interface{}
	assert.NoError(t, yaml.Unmarshal([]byte(got.RawYAML), &m))
	ann := m["metadata"].(map[string]interface{})["annotations"].(map[string]interface{})
	assert.Equal(t, "bar", ann["foo"])
	assert.Equal(t, "keep", ann["helm.sh/resource-policy"])
}

// TestAddKeepAnnotationPreservesKeyOrderAndIndent proves the annotation is
// added without reordering sibling keys or re-indenting the document. The old
// map[string]interface{} round-trip sorted map keys alphabetically and emitted
// 4-space indentation; the yaml.Node round-trip preserves both, so the output
// differs from the input only by the inserted annotation.
func TestAddKeepAnnotationPreservesKeyOrderAndIndent(t *testing.T) {
	r := Resource{RawYAML: `apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: widgets.example.com
spec:
  zeta: 1
  alpha: 2
  group: example.com`}
	got, err := AddKeepAnnotation(r)
	assert.NoError(t, err)

	// Original 2-space indentation is preserved (no 4-space re-indent).
	assert.Contains(t, got.RawYAML, "\n  zeta: 1")
	// Sibling key order under spec is preserved (zeta before alpha before group).
	iZeta := strings.Index(got.RawYAML, "zeta")
	iAlpha := strings.Index(got.RawYAML, "alpha")
	iGroup := strings.Index(got.RawYAML, "group")
	assert.True(t, iZeta < iAlpha && iAlpha < iGroup, "spec key order not preserved: %s", got.RawYAML)
	// Top-level order preserved: apiVersion before kind before metadata before spec.
	assert.True(t,
		strings.Index(got.RawYAML, "apiVersion") <
			strings.Index(got.RawYAML, "kind:") &&
			strings.Index(got.RawYAML, "kind:") <
				strings.Index(got.RawYAML, "spec:"),
		"top-level key order not preserved: %s", got.RawYAML)
	// Annotation present.
	assert.Contains(t, got.RawYAML, "helm.sh/resource-policy: keep")
}

func TestEscapeBraces(t *testing.T) {
	in := `x: "{{ .foo }}" and {{bar}}`
	out := EscapeBraces(in)
	assert.NotContains(t, out, "{{ .foo }}")
	assert.Contains(t, out, `{{ "{{" }}`)
	assert.Contains(t, out, `{{ "}}" }}`)
}
