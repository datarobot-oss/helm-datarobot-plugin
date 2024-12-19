package render_helper

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

// createTestFile creates temporary test files with provided content and returns a function to clean them up.
func createTestFile(t *testing.T, filename, content string) func() {
	err := os.WriteFile(filename, []byte(content), 0644)
	assert.NoError(t, err)

	return func() {
		err := os.Remove(filename)
		assert.NoError(t, err)
	}
}

// TestLoadValuesSingleFile tests loading values from a single file.
func TestRenderChartValuesSingleFile(t *testing.T) {
	valuesFile := "values1.yaml"
	defer createTestFile(t, valuesFile, `
image:
  repository: nginx
  tag: stable
`)()

	values, err := RenderChart("../../testdata/test-chart6/", []string{valuesFile}, []string{})
	assert.NoError(t, err)
	expected := `---
# Source: test-chart6/templates/deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-release-test-chart6
  labels:
    app.kubernetes.io/name: test-chart6
    app.kubernetes.io/instance: test-release
    app.kubernetes.io/managed-by: Helm
spec:
  replicas: 1
  selector:
    matchLabels:
      app.kubernetes.io/name: test-chart6
      app.kubernetes.io/instance: test-release
  template:
    metadata:
      labels:
        app.kubernetes.io/name: test-chart6
        app.kubernetes.io/instance: test-release
        app.kubernetes.io/managed-by: Helm
    spec:
      containers:
        - name: test-chart6
          image: nginx:stable
          resources:
            limits:
              cpu: 100m
              memory: 128Mi
            requests:
              cpu: 100m
              memory: 128Mi
`
	assert.Equal(t, expected, values)
}

// TestRenderChartValuesMultipleFiles tests loading values from multiple files where later files override earlier ones.
func TestRenderChartValuesMultipleFiles(t *testing.T) {
	valuesFile1 := "values1.yaml"
	valuesFile2 := "values2.yaml"

	// Create test data files and defer cleanup
	defer createTestFile(t, valuesFile1, `
image:
  repository: nginx
  tag: stable
`)()
	defer createTestFile(t, valuesFile2, `
image:
  tag: latest
resources:
  limits:
    cpu: 200m
`)()

	values, err := RenderChart("../../testdata/test-chart6/", []string{valuesFile1, valuesFile2}, []string{})
	assert.NoError(t, err)
	expected := `---
# Source: test-chart6/templates/deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-release-test-chart6
  labels:
    app.kubernetes.io/name: test-chart6
    app.kubernetes.io/instance: test-release
    app.kubernetes.io/managed-by: Helm
spec:
  replicas: 1
  selector:
    matchLabels:
      app.kubernetes.io/name: test-chart6
      app.kubernetes.io/instance: test-release
  template:
    metadata:
      labels:
        app.kubernetes.io/name: test-chart6
        app.kubernetes.io/instance: test-release
        app.kubernetes.io/managed-by: Helm
    spec:
      containers:
        - name: test-chart6
          image: nginx:latest
          resources:
            limits:
              cpu: 200m
              memory: 128Mi
            requests:
              cpu: 100m
              memory: 128Mi
`
	assert.Equal(t, expected, values)
}

// TestRenderChartValuesMultipleFilesInputSet tests loading values from multiple files where later files override earlier ones.
func TestRenderChartValuesMultipleFilesInputSet(t *testing.T) {
	valuesFile1 := "values1.yaml"
	valuesFile2 := "values2.yaml"

	// Create test data files and defer cleanup
	defer createTestFile(t, valuesFile1, `
image:
  repository: nginx
  tag: stable
`)()
	defer createTestFile(t, valuesFile2, `
image:
  tag: latest
resources:
  limits:
    cpu: 200m
`)()

	setValues := []string{"replicaCount=3"}
	values, err := RenderChart("../../testdata/test-chart6/", []string{valuesFile1, valuesFile2}, setValues)
	assert.NoError(t, err)
	expected := `---
# Source: test-chart6/templates/deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-release-test-chart6
  labels:
    app.kubernetes.io/name: test-chart6
    app.kubernetes.io/instance: test-release
    app.kubernetes.io/managed-by: Helm
spec:
  replicas: 3
  selector:
    matchLabels:
      app.kubernetes.io/name: test-chart6
      app.kubernetes.io/instance: test-release
  template:
    metadata:
      labels:
        app.kubernetes.io/name: test-chart6
        app.kubernetes.io/instance: test-release
        app.kubernetes.io/managed-by: Helm
    spec:
      containers:
        - name: test-chart6
          image: nginx:latest
          resources:
            limits:
              cpu: 200m
              memory: 128Mi
            requests:
              cpu: 100m
              memory: 128Mi
`
	assert.Equal(t, expected, values)
}

// TestRenderChartEmptyFilesInputSet
func TestRenderChartEmptyFilesInputSet(t *testing.T) {
	setValues := []string{"replicaCount=3", "image.tag=inputset"}
	values, err := RenderChart("../../testdata/test-chart6/", []string{}, setValues)
	assert.NoError(t, err)
	expected := `---
# Source: test-chart6/templates/deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-release-test-chart6
  labels:
    app.kubernetes.io/name: test-chart6
    app.kubernetes.io/instance: test-release
    app.kubernetes.io/managed-by: Helm
spec:
  replicas: 3
  selector:
    matchLabels:
      app.kubernetes.io/name: test-chart6
      app.kubernetes.io/instance: test-release
  template:
    metadata:
      labels:
        app.kubernetes.io/name: test-chart6
        app.kubernetes.io/instance: test-release
        app.kubernetes.io/managed-by: Helm
    spec:
      containers:
        - name: test-chart6
          image: docker.io/alpine/curl:inputset
          resources:
            limits:
              cpu: 100m
              memory: 128Mi
            requests:
              cpu: 100m
              memory: 128Mi
`
	assert.Equal(t, expected, values)
}
