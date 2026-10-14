package manifest

import (
	"testing"

	"github.com/stretchr/testify/assert"
	yaml "gopkg.in/yaml.v3"
)

func TestStripKeepAnnotation(t *testing.T) {
	r := Resource{RawYAML: `kind: CustomResourceDefinition
metadata:
  name: x
  annotations:
    helm.sh/resource-policy: keep
    foo: bar`}
	got, err := StripKeepAnnotation(r)
	assert.NoError(t, err)

	var m map[string]interface{}
	assert.NoError(t, yaml.Unmarshal([]byte(got.RawYAML), &m))
	meta := m["metadata"].(map[string]interface{})
	ann := meta["annotations"].(map[string]interface{})
	_, hasKeep := ann["helm.sh/resource-policy"]
	assert.False(t, hasKeep)
	assert.Equal(t, "bar", ann["foo"])
	assert.Equal(t, "x", meta["name"])
}

func TestStripKeepAnnotationRemovesEmptyAnnotations(t *testing.T) {
	r := Resource{RawYAML: `kind: CustomResourceDefinition
metadata:
  name: x
  annotations:
    helm.sh/resource-policy: keep
spec: {}`}
	got, err := StripKeepAnnotation(r)
	assert.NoError(t, err)

	var m map[string]interface{}
	assert.NoError(t, yaml.Unmarshal([]byte(got.RawYAML), &m))
	meta := m["metadata"].(map[string]interface{})
	_, hasAnnotations := meta["annotations"]
	assert.False(t, hasAnnotations)
	assert.Equal(t, "x", meta["name"])
}

func TestStripKeepAnnotationNoop(t *testing.T) {
	r := Resource{RawYAML: `kind: CustomResourceDefinition
metadata:
  name: x
spec: {}`}
	got, err := StripKeepAnnotation(r)
	assert.NoError(t, err)

	var want, have map[string]interface{}
	assert.NoError(t, yaml.Unmarshal([]byte(r.RawYAML), &want))
	assert.NoError(t, yaml.Unmarshal([]byte(got.RawYAML), &have))
	assert.Equal(t, want, have)
}
