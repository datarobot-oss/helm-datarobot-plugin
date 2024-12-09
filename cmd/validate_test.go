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
		output, err := executeCommand(rootCmd, "validate", "../testdata/test-chart5", "-a", "'custom/images-wrong'")
		assert.Error(t, err)
		expectedOutput := `Error: Image not defined in as imageDoc: busybox:1.36.1`
		assert.Equal(t, expectedOutput, output)
	})

	t.Run("test-chart1", func(t *testing.T) {

		output, err := executeCommand(rootCmd, "validate", "../testdata/test-chart1", "-a", "'datarobot.com/images'")
		assert.NoError(t, err)
		expectedOutput := `Image Doc Valid`
		assert.Equal(t, expectedOutput, output)
	})
}
