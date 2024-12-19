package render_helper

import (
	"os"
	"path"
	"testing"
)

func TestIsChartDirectory(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Test case 1: Directory with Chart.yaml
	chartFilePath := path.Join(tempDir, "Chart.yaml")
	if err := os.WriteFile(chartFilePath, []byte("name: example-chart\nversion: 0.1.0\n"), 0644); err != nil {
		t.Fatalf("Failed to create Chart.yaml: %v", err)
	}

	if !isChartDirectory(tempDir) {
		t.Errorf("Expected true for directory containing Chart.yaml, got false")
	}

	// Test case 2: Directory without Chart.yaml
	emptyDir := path.Join(tempDir, "empty")
	if err := os.Mkdir(emptyDir, 0755); err != nil {
		t.Fatalf("Failed to create empty directory: %v", err)
	}

	if isChartDirectory(emptyDir) {
		t.Errorf("Expected false for directory without Chart.yaml, got true")
	}
}
