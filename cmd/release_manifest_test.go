package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCommandReleaseManifest(t *testing.T) {
	t.Run("Test test-chart1", func(t *testing.T) {
		output, err := executeCommand(rootCmd, "release-manifest ../testdata/test-chart1 -a \"datarobot.com/images\"")
		assert.NoError(t, err)
		// Expected output to compare
		expectedOutput := `images:
  test-image1.tar.zst:
    source: docker.io/datarobotdev/test-image1:1.0.0
    name: docker.io/datarobot/test-image1
    tag: 1.0.0
  test-image2.tar.zst:
    source: docker.io/datarobotdev/test-image2:2.0.0
    name: docker.io/datarobot/test-image2
    tag: 2.0.0
  test-image3.tar.zst:
    source: docker.io/datarobotdev/test-image3:3.0.0
    name: docker.io/datarobot/test-image3
    tag: 3.0.0`
		// Compare the actual output with the expected output
		assert.Equal(t, expectedOutput, output)
	})

	t.Run("Test test-chart4", func(t *testing.T) {
		output, err := executeCommand(rootCmd, "release-manifest ../testdata/test-chart4 -a \"custom/images\"")
		assert.NoError(t, err)
		expectedOutput := `images:
  test-image4.tar.zst:
    source: docker.io/datarobotdev/test-image4:4.0.0
    name: docker.io/datarobot/test-image4
    tag: 4.0.0`
		assert.Equal(t, expectedOutput, output)
	})

	t.Run("Test test-chart4-datarobot", func(t *testing.T) {
		output, err := executeCommand(rootCmd, "release-manifest ../testdata/test-chart4 -a \"datarobot.com/images\"")
		assert.NoError(t, err)
		expectedOutput := `images:
  test-image3.tar.zst:
    source: docker.io/alpine/curl:8.9.1
    name: docker.io/datarobot/curl
    tag: stable
  test-image30.tar.zst:
    source: docker.io/busybox:1.36.1
    name: docker.io/datarobot/busybox
    tag: simple
  test-image31.tar.zst:
    source: docker.io/alpine/curl:8.10.0
    name: docker.io/datarobot/curl
    tag: 8.10.0`

		// Compare the actual output with the expected output
		assert.Equal(t, expectedOutput, output)
	})
	t.Run("Test test-chart4-duplicated", func(t *testing.T) {
		output, err := executeCommand(rootCmd, "release-manifest ../testdata/test-chart4 -a \"custom/images-duplicated\"")
		assert.NoError(t, err)
		// Expected output to compare
		expectedOutput := `images:
  test-image4.tar.zst:
    source: docker.io/datarobotdev/test-image4:4.0.0
    name: docker.io/datarobot/test-image4
    tag: 4.0.0`
		// Compare the actual output with the expected output
		assert.Equal(t, expectedOutput, output)
	})
}
