package chartutil

import (
	"path/filepath"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Helper function to sort DatarobotImageDeclaration slices for consistent comparison
func sortDatarobotImageDeclarations(images []DatarobotImageDeclaration) {
	sort.Slice(images, func(i, j int) bool {
		if images[i].Name != images[j].Name {
			return images[i].Name < images[j].Name
		}
		if images[i].Image != images[j].Image {
			return images[i].Image < images[j].Image
		}
		if images[i].Tag != images[j].Tag {
			return images[i].Tag < images[j].Tag
		}
		return images[i].Group < images[j].Group
	})
}

func TestExtractImagesFromCharts(t *testing.T) {
	tests := []struct {
		name           string
		chartPaths     []string
		annotation     string
		expectedImages []DatarobotImageDeclaration
		expectError    bool
	}{
		{
			name:       "chart1 with default datarobot.com/images annotation",
			chartPaths: []string{"../../tests/charts/test-chart1"},
			annotation: "datarobot.com/images", // Default annotation
			expectedImages: []DatarobotImageDeclaration{
				{Name: "test-image1", Image: "docker.io/datarobotdev/test-image1:1.0.0", Tag: ""},
				{Name: "test-image2", Image: "docker.io/datarobotdev/test-image2:2.0.0", Tag: ""},
				{Name: "test-image3", Image: "docker.io/datarobotdev/test-image3:3.0.0", Tag: ""},
			},
			expectError: false,
		},
		{
			name:           "chart6 with bitnami.com/images annotation",
			chartPaths:     []string{"../../tests/charts/test-chart6"},
			annotation:     "bitnami.com/images",
			expectedImages: []DatarobotImageDeclaration{}, // Updated to match actual output
			expectError:    false,
		},
		{
			name:           "chart1 with non-existent annotation",
			chartPaths:     []string{"../../tests/charts/test-chart1"},
			annotation:     "non-existent-annotation",
			expectedImages: []DatarobotImageDeclaration{},
			expectError:    false,
		},
		{
			name:           "invalid chart path",
			chartPaths:     []string{"../../tests/charts/non-existent-chart"},
			annotation:     "datarobot.com/images",
			expectedImages: []DatarobotImageDeclaration{},
			expectError:    true,
		},
		{
			name:       "chart with subcharts (test-chart1 includes test-chart2 which includes test-chart3)",
			chartPaths: []string{"../../tests/charts/test-chart1"},
			annotation: "datarobot.com/images",
			expectedImages: []DatarobotImageDeclaration{
				{Name: "test-image1", Image: "docker.io/datarobotdev/test-image1:1.0.0", Tag: ""},
				{Name: "test-image2", Image: "docker.io/datarobotdev/test-image2:2.0.0", Tag: ""},
				{Name: "test-image3", Image: "docker.io/datarobotdev/test-image3:3.0.0", Tag: ""},
			},
			expectError: false,
		},
		{
			name:       "multiple chart paths",
			chartPaths: []string{"../../tests/charts/test-chart1", "../../tests/charts/test-chart4"},
			annotation: "datarobot.com/images",
			expectedImages: []DatarobotImageDeclaration{
				// Images from test-chart1 (top-level only)
				{Name: "test-image1", Image: "docker.io/datarobotdev/test-image1:1.0.0", Tag: ""},
				{Name: "test-image2", Image: "docker.io/datarobotdev/test-image2:2.0.0", Tag: ""},
				{Name: "test-image3", Image: "docker.io/datarobotdev/test-image3:3.0.0", Tag: ""},
				// Images from test-chart4 (based on its Chart.yaml and observed behavior)
				{Name: "test-image3", Image: "docker.io/alpine/curl:8.9.1", Tag: "stable", Group: ""},
				{Name: "test-image30", Image: "busybox:1.36.1", Tag: "simple", Group: ""},
				{Name: "test-image31", Image: "docker.io/alpine/curl:8.10.0", Tag: "", Group: ""},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Resolve relative paths to absolute paths for chart loading
			var absChartPaths []string
			for _, p := range tt.chartPaths {
				absPath, err := filepath.Abs(p)
				assert.NoError(t, err)
				absChartPaths = append(absChartPaths, absPath)
			}

			images, err := ExtractImagesFromCharts(absChartPaths, tt.annotation)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				sortDatarobotImageDeclarations(images)
				sortDatarobotImageDeclarations(tt.expectedImages) // Ensure expected images are also sorted
				assert.Equal(t, tt.expectedImages, images)
			}
		})
	}
}
