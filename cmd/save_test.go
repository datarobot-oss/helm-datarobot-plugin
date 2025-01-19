package cmd

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCommandSave(t *testing.T) {

	t.Run("test-chart4-dry-run", func(t *testing.T) {

		output, err := executeCommand(rootCmd, "save ../testdata/test-chart4 -a custom/images-duplicated --dry-run --output test.tgz -a \"datarobot.com/images\"")
		assert.NoError(t, err)
		expectedOutput := `[Dry-Run] Pulling image: docker.io/alpine/curl:8.9.1
[Dry-Run] ReTagging image: docker.io/alpine/curl:8.9.1 > docker.io/alpine/curl:stable
[Dry-Run] adding image to tgz: alpine/curl:stable.tgz
[Dry-Run] Pulling image: docker.io/busybox:1.36.1
[Dry-Run] ReTagging image: docker.io/busybox:1.36.1 > docker.io/busybox:simple
[Dry-Run] adding image to tgz: busybox:simple.tgz
[Dry-Run] Pulling image: docker.io/alpine/curl:8.10.0
[Dry-Run] adding image to tgz: alpine/curl:8.10.0.tgz
[Dry-Run] Tarball created successfully: test.tgz`
		assert.Equal(t, expectedOutput, output)
	})

	t.Run("duplicated", func(t *testing.T) {

		filePath := "image-test.tgz"
		output, err := executeCommand(rootCmd, "save ../testdata/test-chart5 -a custom/images-duplicated --dry-run=false --output "+filePath)
		assert.NoError(t, err)
		expectedOutput := `Pulling image: docker.io/alpine/curl:8.9.1
Pulling image: docker.io/alpine/curl:8.9.1
 archive alpine/curl:8.9.1.tgz already exists
Tarball created successfully: image-test.tgz`

		assert.Equal(t, expectedOutput, output)
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			t.Errorf("File was not created: %s", filePath)
		}
		err = os.Remove(filePath)
		if err != nil {
			t.Fatalf("Failed to remove file: %v", err)
		}
	})

	t.Run("test-chart4", func(t *testing.T) {

		filePath := "image-test.tgz"
		output, err := executeCommand(rootCmd, "save ../testdata/test-chart4 --dry-run=false -a datarobot.com/images --output "+filePath)
		assert.NoError(t, err)

		// Expected output to compare
		expectedOutput := `Pulling image: docker.io/alpine/curl:8.9.1
ReTagging image: docker.io/alpine/curl:8.9.1 > docker.io/alpine/curl:stable
Pulling image: docker.io/busybox:1.36.1
ReTagging image: docker.io/busybox:1.36.1 > docker.io/busybox:simple
Pulling image: docker.io/alpine/curl:8.10.0
Tarball created successfully: image-test.tgz`

		assert.Equal(t, expectedOutput, output)

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
