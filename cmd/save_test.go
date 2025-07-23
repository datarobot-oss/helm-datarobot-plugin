package cmd

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

const SAVE_TEST_ARCHIVE = "image-load.tar.zst"

func TestCommandSave(t *testing.T) {

	t.Run("dry-run", func(t *testing.T) {

		output, err := executeCommand(rootCmd, "save ../tests/charts/test-chart4 -a custom/images-duplicated --dry-run --output "+SAVE_TEST_ARCHIVE+" -a datarobot.com/images")
		assert.NoError(t, err)
		expectedOutput := `[Dry-Run] Pulling image: docker.io/alpine/curl:8.9.1
[Dry-Run] ReTagging image: docker.io/alpine/curl:8.9.1 > docker.io/alpine/curl:stable
[Dry-Run] Pulling image: docker.io/busybox:1.36.1
[Dry-Run] ReTagging image: docker.io/busybox:1.36.1 > docker.io/busybox:simple
[Dry-Run] Pulling image: docker.io/alpine/curl:8.10.0
[Dry-Run] Tarball created successfully: ` + SAVE_TEST_ARCHIVE
		assert.Equal(t, expectedOutput, output)
	})

	t.Run("duplicated", func(t *testing.T) {

		output, err := executeCommand(rootCmd, "save ../tests/charts/test-chart5 -a custom/images-duplicated --dry-run=false --output "+SAVE_TEST_ARCHIVE)
		assert.NoError(t, err)
		expectedOutput := `Pulling image: docker.io/alpine/curl:8.9.1
Pulling image: docker.io/alpine/curl:8.9.1
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

	t.Run("full", func(t *testing.T) {
		output, err := executeCommand(rootCmd, "save ../tests/charts/test-chart4 -a datarobot.com/images --output "+SAVE_TEST_ARCHIVE)
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
	t.Run("layers", func(t *testing.T) {
		output, err := executeCommand(rootCmd, "save ../tests/charts/test-chart6 -a layers --output "+SAVE_TEST_ARCHIVE)
		assert.NoError(t, err)

		expectedOutput := `Pulling image: docker.io/nginx:1.27.4-alpine
ReTagging image: docker.io/nginx:1.27.4-alpine > docker.io/nginx:simple
Pulling image: docker.io/nginx:1.27.4-alpine3.21
Pulling image: docker.io/nginx:1.27-alpine3.21
Tarball created successfully: ` + SAVE_TEST_ARCHIVE

		assert.Equal(t, expectedOutput, output)

		// Check if the file exists
		if os.IsNotExist(err) {
			t.Errorf("File was not created: %s", SAVE_TEST_ARCHIVE)
		}

		// Clean up: remove the file after the test
		err = os.Remove(SAVE_TEST_ARCHIVE)
		if err != nil {
			t.Fatalf("Failed to remove file: %v", err)
		}
	})

	t.Run("skip-image-group", func(t *testing.T) {
		output, err := executeCommand(rootCmd, "save ../tests/charts/test-chart6 --dry-run -a image/groups --skip-group test1 --skip-group test2 --output "+SAVE_TEST_ARCHIVE)
		assert.NoError(t, err)
		expectedOutput := `Skipping image: docker.io/alpine/curl:8.9.10

Skipping image: docker.io/alpine/curl:8.9.11

Skipping image: docker.io/alpine/curl:8.9.2

[Dry-Run] Pulling image: docker.io/alpine/curl:8.9.3
[Dry-Run] Tarball created successfully: ` + SAVE_TEST_ARCHIVE

		assert.Equal(t, expectedOutput, output)
	})

	t.Run("wrong-level", func(t *testing.T) {
		output, err := executeCommand(rootCmd, "save ../tests/charts/test-chart4 --level=wrong")
		assert.Error(t, err)
		expectedOutput := `Error: Invalid compression level. Available options: fastest, default, better, best`
		assert.Equal(t, expectedOutput, output)
	})
}
