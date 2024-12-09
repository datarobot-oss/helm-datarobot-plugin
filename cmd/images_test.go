package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCommandImages(t *testing.T) {
	output, err := executeCommand(rootCmd, "image", "../testdata/test-chart1")
	assert.NoError(t, err)
	expectedOutput := `- name: test-image1
  image: docker.io/datarobotdev/test-image1:1.0.0
- name: test-image2
  image: docker.io/datarobotdev/test-image2:2.0.0
- name: test-image3
  image: docker.io/datarobotdev/test-image3:3.0.0`

	// Compare the actual output with the expected output
	assert.Equal(t, expectedOutput, output)
}
