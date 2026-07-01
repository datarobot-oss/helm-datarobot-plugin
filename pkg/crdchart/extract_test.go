package crdchart

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const sampleManifest = `---
# Source: c/templates/crd-a.yaml
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: widgets.example.com
spec: {}
---
# Source: c/templates/deploy.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: web
  namespace: app
---
# Source: c/templates/cr.yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: reader
`

func TestParseManifests(t *testing.T) {
	res, err := ParseManifests(sampleManifest)
	assert.NoError(t, err)
	assert.Len(t, res, 3)
	assert.Equal(t, "CustomResourceDefinition", res[0].Kind)
	assert.Equal(t, "widgets.example.com", res[0].Name)
	assert.Equal(t, "Deployment", res[1].Kind)
	assert.Equal(t, "app", res[1].Namespace)
	assert.Contains(t, res[0].RawYAML, "kind: CustomResourceDefinition")
}

func TestParseManifestsSkipsEmptyDocs(t *testing.T) {
	res, err := ParseManifests("---\n\n---\n# comment only\n---\n{}\n")
	assert.NoError(t, err)
	assert.Len(t, res, 0)
}

// TestParseManifestsSeparatorWithTrailingComment proves the splitter treats a
// "---" separator that carries a trailing comment (e.g. "--- # Source: ...")
// as a real document break. The naive strings.Split(manifest, "\n---\n")
// requires the separator line to be exactly "---" and therefore FAILS to split
// here, gluing the two documents into one and losing the second resource's
// kind. The line-anchored regex splitter handles it.
func TestParseManifestsSeparatorWithTrailingComment(t *testing.T) {
	manifest := `--- # Source: c/templates/crd-a.yaml
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: widgets.example.com
spec: {}
--- # Source: c/templates/deploy.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: web
  namespace: app
`
	res, err := ParseManifests(manifest)
	assert.NoError(t, err)
	assert.Len(t, res, 2)
	assert.Equal(t, "CustomResourceDefinition", res[0].Kind)
	assert.Equal(t, "widgets.example.com", res[0].Name)
	assert.Equal(t, "Deployment", res[1].Kind)
	assert.Equal(t, "web", res[1].Name)
}

// TestParseManifestsSeparatorWithTrailingSpaces proves a separator line with
// trailing whitespace ("---  ") is still recognized as a document break. The
// naive splitter requires an exact "\n---\n" and would glue the documents.
func TestParseManifestsSeparatorWithTrailingSpaces(t *testing.T) {
	manifest := "apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: a\n---  \napiVersion: v1\nkind: Secret\nmetadata:\n  name: b\n"
	res, err := ParseManifests(manifest)
	assert.NoError(t, err)
	assert.Len(t, res, 2)
	assert.Equal(t, "ConfigMap", res[0].Kind)
	assert.Equal(t, "Secret", res[1].Kind)
}

// TestParseManifestsIgnoresIndentedTripleDash confirms an indented "---" inside
// a block scalar is never treated as a document separator (only column-0 "---").
func TestParseManifestsIgnoresIndentedTripleDash(t *testing.T) {
	manifest := `apiVersion: v1
kind: ConfigMap
metadata:
  name: cm
data:
  note: |
    line
    ---
    still same doc
`
	res, err := ParseManifests(manifest)
	assert.NoError(t, err)
	assert.Len(t, res, 1)
	assert.Equal(t, "ConfigMap", res[0].Kind)
	assert.Contains(t, res[0].RawYAML, "still same doc")
}

func TestKeepMatchingIsCRD(t *testing.T) {
	res, _ := ParseManifests(sampleManifest)
	crds := KeepMatching(res, IsCRD)
	assert.Len(t, crds, 1)
	assert.Equal(t, "widgets.example.com", crds[0].Name)
}

func TestDedupeByName(t *testing.T) {
	res := []Resource{
		{Kind: "CustomResourceDefinition", Name: "a", RawYAML: "v1"},
		{Kind: "CustomResourceDefinition", Name: "b", RawYAML: "b"},
		{Kind: "CustomResourceDefinition", Name: "a", RawYAML: "v2"},
	}
	kept, collisions := DedupeByName(res)
	assert.Len(t, kept, 2)
	assert.Equal(t, []string{"a"}, collisions)
	// last wins
	var aRaw string
	for _, r := range kept {
		if r.Name == "a" {
			aRaw = r.RawYAML
		}
	}
	assert.Equal(t, "v2", aRaw)
}
