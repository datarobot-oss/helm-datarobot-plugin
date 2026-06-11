package render_helper

import (
	"os"
	"path/filepath"
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

	values, err := RenderChart("../../tests/charts/test-chart6/", []string{valuesFile}, []string{}, nil)
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

	values, err := RenderChart("../../tests/charts/test-chart6/", []string{valuesFile1, valuesFile2}, []string{}, nil)
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
	values, err := RenderChart("../../tests/charts/test-chart6/", []string{valuesFile1, valuesFile2}, setValues, nil)
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
	values, err := RenderChart("../../tests/charts/test-chart6/", []string{}, setValues, nil)
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

// TestRenderChartWithOptions tests that custom RenderOptions are applied correctly.
func TestRenderChartWithOptions(t *testing.T) {
	opts := &RenderOptions{
		Namespace:   "custom-ns",
		ReleaseName: "custom-release",
		KubeVersion: "v1.29.0",
		IncludeCRDs: false,
		APIVersions: []string{"apps/v1"},
	}

	setValues := []string{"image.tag=custom"}
	result, err := RenderChart("../../tests/charts/test-chart6/", []string{}, setValues, opts)
	assert.NoError(t, err)
	// Release name from opts reflected in rendered output.
	assert.Contains(t, result, "name: custom-release-test-chart6")
	assert.Contains(t, result, "app.kubernetes.io/instance: custom-release")
}

// TestRenderChart_IncludeHooks verifies that hook manifests appear only when IncludeHooks=true.
// Uses an inline minimal chart written to t.TempDir() — does NOT modify tests/charts/.
func TestRenderChart_IncludeHooks(t *testing.T) {
	chartDir := t.TempDir()

	// Chart.yaml
	chartYAML := `apiVersion: v2
name: hook-test
version: 0.1.0
`
	if err := os.WriteFile(filepath.Join(chartDir, "Chart.yaml"), []byte(chartYAML), 0644); err != nil {
		t.Fatalf("write Chart.yaml: %v", err)
	}

	// templates/
	templatesDir := filepath.Join(chartDir, "templates")
	if err := os.MkdirAll(templatesDir, 0755); err != nil {
		t.Fatalf("mkdir templates: %v", err)
	}

	// Normal resource
	normalCM := `apiVersion: v1
kind: ConfigMap
metadata:
  name: normal-cm
data:
  key: value
`
	if err := os.WriteFile(filepath.Join(templatesDir, "configmap.yaml"), []byte(normalCM), 0644); err != nil {
		t.Fatalf("write configmap.yaml: %v", err)
	}

	// Hook resource
	hookCM := `apiVersion: v1
kind: ConfigMap
metadata:
  name: hook-cm
  annotations:
    "helm.sh/hook": pre-install
    "helm.sh/hook-delete-policy": hook-succeeded
data:
  key: hook-value
`
	if err := os.WriteFile(filepath.Join(templatesDir, "hook-configmap.yaml"), []byte(hookCM), 0644); err != nil {
		t.Fatalf("write hook-configmap.yaml: %v", err)
	}

	// Without hooks: hook-cm must NOT appear.
	resultNoHooks, err := RenderChart(chartDir, []string{}, []string{}, &RenderOptions{
		Namespace:    "test",
		ReleaseName:  "test",
		KubeVersion:  "v1.27.0",
		IncludeHooks: false,
	})
	assert.NoError(t, err)
	assert.Contains(t, resultNoHooks, "normal-cm", "normal resource must appear")
	assert.NotContains(t, resultNoHooks, "hook-cm", "hook resource must not appear when IncludeHooks=false")

	// With hooks: hook-cm must appear.
	resultWithHooks, err := RenderChart(chartDir, []string{}, []string{}, &RenderOptions{
		Namespace:    "test",
		ReleaseName:  "test",
		KubeVersion:  "v1.27.0",
		IncludeHooks: true,
	})
	assert.NoError(t, err)
	assert.Contains(t, resultWithHooks, "normal-cm", "normal resource must appear")
	assert.Contains(t, resultWithHooks, "hook-cm", "hook resource must appear when IncludeHooks=true")
}
