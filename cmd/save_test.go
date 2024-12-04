package cmd

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCommandSave(t *testing.T) {

	t.Run("test-chart4-dry-run", func(t *testing.T) {
		// Capture the output
		var stdoutBuf bytes.Buffer
		rootCmd.SetOut(io.Writer(&stdoutBuf))

		// Set arguments for the command (simulate CLI input)
		arg := "save ../testdata/test-chart4 --output test.tgz --dry-run"
		rootCmd.SetArgs(strings.Fields(arg))

		// Execute command while capturing output
		err := rootCmd.Execute()
		assert.NoError(t, err)

		// Expected output to compare
		expectedOutput := `[Dry-Run] Pulling image: docker.io/alpine/curl:8.9.1
[Dry-Run] ReTagging image: docker.io/alpine/curl:8.9.1 > docker.io/alpine/curl:stable
[Dry-Run] adding image to tgz: curl:stable.tgz
[Dry-Run] Pulling image: docker.io/busybox:1.36.1
[Dry-Run] ReTagging image: docker.io/busybox:1.36.1 > docker.io/busybox:simple
[Dry-Run] adding image to tgz: busybox:simple.tgz
[Dry-Run] Pulling image: docker.io/alpine/curl:8.10.0
[Dry-Run] adding image to tgz: curl:8.10.0.tgz
[Dry-Run] Tarball created successfully: test.tgz
`

		// Compare the actual output with the expected output
		assert.Equal(t, expectedOutput, stdoutBuf.String())
	})

	t.Run("test-chart4", func(t *testing.T) {
		// Capture the output
		var stdoutBuf bytes.Buffer
		rootCmd.SetOut(io.Writer(&stdoutBuf))
		filePath := "image-test.tgz"
		// Set arguments for the command (simulate CLI input)
		arg := "save ../testdata/test-chart4 --dry-run=false --output " + filePath
		rootCmd.SetArgs(strings.Fields(arg))

		// Execute command while capturing output
		err := rootCmd.Execute()
		assert.NoError(t, err)

		// Expected output to compare
		expectedOutput := `Pulling image: docker.io/alpine/curl:8.9.1
ReTagging image: docker.io/alpine/curl:8.9.1 > docker.io/alpine/curl:stable
Pulling image: docker.io/busybox:1.36.1
ReTagging image: docker.io/busybox:1.36.1 > docker.io/busybox:simple
Pulling image: docker.io/alpine/curl:8.10.0
Tarball created successfully: image-test.tgz
`

		// Compare the actual output with the expected output
		assert.Equal(t, expectedOutput, stdoutBuf.String())

		// Check if the file exists
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			t.Errorf("File was not created: %s", filePath)
		}

		// Clean up: remove the file after the test
		err = os.Remove(filePath)
		if err != nil {
			t.Fatalf("Failed to remove file: %v", err)
		}
	})

}
