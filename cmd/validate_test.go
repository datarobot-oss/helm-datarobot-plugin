package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCommandValidate(t *testing.T) {
	t.Run("test-chart5", func(t *testing.T) {

		output, err := executeCommand(rootCmd, "validate", "../testdata/test-chart5")
		assert.NoError(t, err)
		expectedOutput := `Image Doc Valid`
		assert.Equal(t, expectedOutput, output)
	})
	t.Run("test-chart5/error", func(t *testing.T) {
		_, err := executeCommand(rootCmd, "validate", "../testdata/test-chart5", "-a", "'custom/images-wrong'")
		assert.Error(t, err)
		// TODO rendering of the files is random, thefore is very difficult to predict which image was not defined
		// TODO order the result of the rendered files
		// expectedOutput := `Error: Image not defined in as imageDoc: docker.io/alpine/curl:8.9.1`
		// assert.Equal(t, expectedOutput, output)
	})

	t.Run("test-chart1", func(t *testing.T) {
		output, err := executeCommand(rootCmd, "validate", "../testdata/test-chart1", "-a", "'datarobot.com/images'", "--debug")
		assert.NoError(t, err)
		expectedOutput := `Image Doc Valid`
		assert.Equal(t, expectedOutput, output)
	})
}
