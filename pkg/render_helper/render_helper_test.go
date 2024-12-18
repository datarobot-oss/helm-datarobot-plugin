package render_helper

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestLoadValuesSingleFile tests loading values from a single file.
func TestLoadValuesSingleFile(t *testing.T) {
	valuesFile := "testdata/values1.yaml"

	// Create test data files
	defer setupTestFiles(t, map[string]string{
		valuesFile: `
replicaCount: 1
image:
  repository: nginx
  tag: stable
`,
	})()

	values, err := loadValues([]string{valuesFile}, []string{})
	assert.NoError(t, err)

	expected := map[string]interface{}{
		"replicaCount": 1,
		"image": map[string]interface{}{
			"repository": "nginx",
			"tag":        "stable",
		},
	}
	assert.Equal(t, expected, values)
}

// TestLoadValuesMultipleFiles tests loading values from multiple files.
func TestLoadValuesMultipleFiles(t *testing.T) {
	valuesFile1 := "testdata/values1.yaml"
	valuesFile2 := "testdata/values2.yaml"

	// Create test data files
	defer setupTestFiles(t, map[string]string{
		valuesFile1: `
replicaCount: 1
image:
  repository: nginx
  tag: stable
`,
		valuesFile2: `
image:
  tag: latest
resources:
  limits:
    cpu: 200m
`,
	})()

	values, err := loadValues([]string{valuesFile1, valuesFile2}, []string{})
	assert.NoError(t, err)

	expected := map[string]interface{}{
		"replicaCount": 1,
		"image": map[string]interface{}{
			"repository": "nginx",
			"tag":        "latest",
		},
		"resources": map[string]interface{}{
			"limits": map[string]interface{}{
				"cpu": "200m",
			},
		},
	}
	assert.Equal(t, expected, values)
}

// TestLoadValuesWithSet tests loading values from files and overriding them with set values.
func TestLoadValuesWithSet(t *testing.T) {
	valuesFile1 := "testdata/values1.yaml"
	valuesFile2 := "testdata/values2.yaml"

	// Create test data files
	defer setupTestFiles(t, map[string]string{
		valuesFile1: `
replicaCount: 1
image:
  repository: nginx
  tag: stable
`,
		valuesFile2: `
image:
  tag: latest
resources:
  limits:
    cpu: 200m
`,
	})()

	values, err := loadValues([]string{valuesFile1, valuesFile2}, []string{"replicaCount=3", "resources.limits.memory=512Mi"})
	assert.NoError(t, err)

	expected := map[string]interface{}{
		"replicaCount": 3,
		"image": map[string]interface{}{
			"repository": "nginx",
			"tag":        "latest",
		},
		"resources": map[string]interface{}{
			"limits": map[string]interface{}{
				"cpu":    "200m",
				"memory": "512Mi",
			},
		},
	}
	assert.Equal(t, expected, values)
}

// setupTestFiles creates temporary test data files and cleans them up after testing.
func setupTestFiles(t *testing.T, files map[string]string) func() {
	for filename, content := range files {
		err := os.MkdirAll("testdata", 0755)
		assert.NoError(t, err)
		err = os.WriteFile(filename, []byte(content), 0644)
		assert.NoError(t, err)
	}
	return func() {
		for filename := range files {
			err := os.Remove(filename)
			assert.NoError(t, err)
		}
		err := os.RemoveAll("testdata")
		assert.NoError(t, err)
	}
}
