package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCommandGenerate(t *testing.T) {
	t.Run("test-chart5", func(t *testing.T) {
		output, err := executeCommand(rootCmd, "generate", "../testdata/test-chart5")
		assert.NoError(t, err)
		expectedOutput := `annotations:
  datarobot.com/images: |
    - name: curl_891
      image: docker.io/alpine/curl:8.9.1
    - name: busybox_1361
      image: busybox:1.36.1`
		assert.Equal(t, expectedOutput, output)
	})
}
