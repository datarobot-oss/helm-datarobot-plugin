package cmd

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

const SAVE_TEST_ARCHIVE = "image-load.tar.zst"

func TestCommandSave(t *testing.T) {

	t.Run("test-chart4-dry-run", func(t *testing.T) {

		output, err := executeCommand(rootCmd, "save ../tests/charts/test-chart4 -a custom/images-duplicated --dry-run --output "+SAVE_TEST_ARCHIVE+" -a \"datarobot.com/images\"")
		assert.NoError(t, err)
		expectedOutput := `[Dry-Run] Pulling image: docker.io/alpine/curl:8.9.1
[Dry-Run] ReTagging image: docker.io/alpine/curl:8.9.1 > docker.io/alpine/curl:stable
[Dry-Run] adding image to tgz: alpine/curl:stable.tgz
[Dry-Run] Pulling image: docker.io/busybox:1.36.1
[Dry-Run] ReTagging image: docker.io/busybox:1.36.1 > docker.io/busybox:simple
[Dry-Run] adding image to tgz: busybox:simple.tgz
[Dry-Run] Pulling image: docker.io/alpine/curl:8.10.0
[Dry-Run] adding image to tgz: alpine/curl:8.10.0.tgz
[Dry-Run] Tarball created successfully: ` + SAVE_TEST_ARCHIVE
		assert.Equal(t, expectedOutput, output)
	})

	t.Run("duplicated", func(t *testing.T) {

		output, err := executeCommand(rootCmd, "save ../tests/charts/test-chart5 -a custom/images-duplicated --dry-run=false --output "+SAVE_TEST_ARCHIVE)
		assert.NoError(t, err)
		expectedOutput := `Pulling image: docker.io/alpine/curl:8.9.1
Pulling image: docker.io/alpine/curl:8.9.1
 archive alpine/curl:8.9.1.tgz already exists
Tarball created successfully: ` + SAVE_TEST_ARCHIVE

		assert.Equal(t, expectedOutput, output)
		if _, err := os.Stat(SAVE_TEST_ARCHIVE); os.IsNotExist(err) {
			t.Errorf("File was not created: %s", SAVE_TEST_ARCHIVE)
		}
		err = os.Remove(SAVE_TEST_ARCHIVE)
		if err != nil {
			t.Fatalf("Failed to remove file: %v", err)
		}
	})

	t.Run("test-chart4", func(t *testing.T) {
		output, err := executeCommand(rootCmd, "save ../tests/charts/test-chart4 --dry-run=false -a datarobot.com/images --output "+SAVE_TEST_ARCHIVE)
		assert.NoError(t, err)

		expectedOutput := `Pulling image: docker.io/alpine/curl:8.9.1
ReTagging image: docker.io/alpine/curl:8.9.1 > docker.io/alpine/curl:stable
Pulling image: docker.io/busybox:1.36.1
ReTagging image: docker.io/busybox:1.36.1 > docker.io/busybox:simple
Pulling image: docker.io/alpine/curl:8.10.0
Tarball created successfully: ` + SAVE_TEST_ARCHIVE

		assert.Equal(t, expectedOutput, output)

		// Check if the file exists
		if _, err := os.Stat(SAVE_TEST_ARCHIVE); os.IsNotExist(err) {
			t.Errorf("File was not created: %s", SAVE_TEST_ARCHIVE)
		}

		// Clean up: remove the file after the test
		err = os.Remove(SAVE_TEST_ARCHIVE)
		if err != nil {
			t.Fatalf("Failed to remove file: %v", err)
		}
	})
	t.Run("wrong-level", func(t *testing.T) {
		output, err := executeCommand(rootCmd, "save ../tests/charts/test-chart4 --level=wrong")
		assert.Error(t, err)
		expectedOutput := `Error: Invalid compression level. Available options: fastest, default, better, best`
		assert.Equal(t, expectedOutput, output)
	})

}
