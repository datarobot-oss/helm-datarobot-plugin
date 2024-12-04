package cmd

import (
	"bytes"
	"io"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

// TestYAMLCmdWithSingleFile tests the yaml command with a single file and compares the output
func TestCommandImages(t *testing.T) {
	// Create a new root command for testing
	rootCmd := &cobra.Command{Use: "image"}
	rootCmd.AddCommand(imageCmd)

	// Capture the output
	var stdoutBuf bytes.Buffer
	rootCmd.SetOut(io.Writer(&stdoutBuf))

	// Set arguments for the command (simulate CLI input)
	rootCmd.SetArgs([]string{"image", "../testdata/test-chart1"})

	// Execute command while capturing output
	err := rootCmd.Execute()
	assert.NoError(t, err)

	// Expected output to compare
	expectedOutput := `- name: test-image1
  image: docker.io/datarobotdev/test-image1:1.0.0
- name: test-image2
  image: docker.io/datarobotdev/test-image2:2.0.0
- name: test-image3
  image: docker.io/datarobotdev/test-image3:3.0.0
`

	// Compare the actual output with the expected output
	assert.Equal(t, expectedOutput, stdoutBuf.String())
}
