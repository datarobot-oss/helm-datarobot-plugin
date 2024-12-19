package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCommandValidate(t *testing.T) {
	t.Run("test-chart1", func(t *testing.T) {
		output, err := executeCommand(rootCmd, "validate ../testdata/test-chart1 -a \"datarobot.com/images\"")
		assert.NoError(t, err)
		expectedOutput := `Image Doc Valid`
		assert.Equal(t, expectedOutput, output)
	})
	t.Run("test-chart5", func(t *testing.T) {
		output, err := executeCommand(rootCmd, "validate ../testdata/test-chart5")
		assert.NoError(t, err)
		expectedOutput := `Image Doc Valid`
		assert.Equal(t, expectedOutput, output)
	})
	t.Run("test-chart5/error", func(t *testing.T) {
		output, err := executeCommand(rootCmd, "validate ../testdata/test-chart5 -a \"custom/images-wrong\"")
		assert.Error(t, err)
		expectedOutput := "Error: Images not declared as ImageDoc:\nbusybox:1.36.1\ndocker.io/alpine/curl:8.9.1"
		assert.Equal(t, expectedOutput, output)
	})
	t.Run("test-chart5/empty", func(t *testing.T) {
		output, err := executeCommand(rootCmd, "validate ../testdata/test-chart5 -a \"custom/non-existing\"")
		assert.Error(t, err)
		expectedOutput := `Error: imageDoc is empty`
		assert.Equal(t, expectedOutput, output)
	})
	t.Run("test-chart3/override", func(t *testing.T) {
		output, err := executeCommand(rootCmd, "validate ../testdata/test-chart3 -a custom.com/images-override --set image.repository=docker.io/testoverride/test-image3")
		assert.NoError(t, err)
		expectedOutput := `Image Doc Valid`
		assert.Equal(t, expectedOutput, output)
	})

}
