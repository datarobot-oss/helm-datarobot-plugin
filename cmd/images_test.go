package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCommandImages(t *testing.T) {
	t.Run("base", func(t *testing.T) {
		output, err := executeCommand(rootCmd, "image ../tests/charts/test-chart1")
		assert.NoError(t, err)
		expectedOutput := `- name: test-image1
  image: docker.io/datarobotdev/test-image1:1.0.0
- name: test-image2
  image: docker.io/datarobotdev/test-image2:2.0.0
- name: test-image3
  image: docker.io/datarobotdev/test-image3:3.0.0`

		// Compare the actual output with the expected output
		assert.Equal(t, expectedOutput, output)
	})

	t.Run("upgrade-from-match", func(t *testing.T) {
		output, err := executeCommand(rootCmd, "image ../tests/charts/test-chart7 -a datarobot.com/images --upgrade-from 10.0")
		assert.NoError(t, err)
		assert.Contains(t, output, "docker.io/alpine/curl:8.10.0")
	})

	t.Run("upgrade-from-nomatch", func(t *testing.T) {
		output, err := executeCommand(rootCmd, "image ../tests/charts/test-chart7 -a datarobot.com/images --upgrade-from 11.0")
		assert.NoError(t, err)
		assert.NotContains(t, output, "docker.io/alpine/curl:8.10.0")
	})

	t.Run("no-upgrade", func(t *testing.T) {
		output, err := executeCommand(rootCmd, "image ../tests/charts/test-chart7 -a datarobot.com/images --no-upgrade")
		assert.NoError(t, err)
		assert.NotContains(t, output, "Pulling image: docker.io/alpine/curl:8.10.0")
	})

}
