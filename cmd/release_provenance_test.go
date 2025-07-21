package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCommandReleaseProvenance(t *testing.T) {
	t.Run("test-chart6 with default annotation", func(t *testing.T) {
		output, err := executeCommand(rootCmd, "release-provenance ../tests/charts/test-chart6")
		assert.NoError(t, err)
		expectedOutput := `[
  {
    "image": "docker.io/alpine/curl:8.9.1",
    "repo": "",
    "commit": "",
  },
]`
		assert.Equal(t, expectedOutput, output)
	})

	t.Run("invalid chart path", func(t *testing.T) {
		output, err := executeCommand(rootCmd, "release-provenance non-existent-chart.tgz")
		assert.Error(t, err)
		expectedOutput := `Error: error ExtractImagesFromCharts: Error loading chart non-existent-chart.tgz: stat non-existent-chart.tgz: no such file or directory
Usage:
  helm-datarobot release-provenance [flags]

Flags:
  -h, --help   help for release-provenance
`
		assert.Equal(t, expectedOutput, output)
	})
}
