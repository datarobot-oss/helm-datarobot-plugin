package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCommandGenerate(t *testing.T) {
	t.Run("test-chart5", func(t *testing.T) {
		output, err := executeCommand(rootCmd, "generate ../testdata/test-chart5")
		assert.NoError(t, err)
		expectedOutput := `annotations:
  datarobot.com/images: |
    - name: busybox_1361
      image: busybox:1.36.1
    - name: curl_891
      image: docker.io/alpine/curl:8.9.1`
		assert.Equal(t, expectedOutput, output)
	})

	t.Run("test-chart1", func(t *testing.T) {
		output, err := executeCommand(rootCmd, "generate ../testdata/test-chart1")
		assert.NoError(t, err)
		expectedOutput := `annotations:
  datarobot.com/images: |
    - name: test-image1_100
      image: docker.io/datarobotdev/test-image1:1.0.0
    - name: test-image2_200
      image: docker.io/datarobotdev/test-image2:2.0.0
    - name: test-image3_300
      image: docker.io/datarobotdev/test-image3:3.0.0`
		assert.Equal(t, expectedOutput, output)
	})
}
