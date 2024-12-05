package cmd

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCommandLoad(t *testing.T) {

	t.Run("test-chart", func(t *testing.T) {
		filePath := "image-load.tgz"
		output, err := executeCommand(rootCmd, "save", "../testdata/test-chart4", "-a", "custom/loadimages", "--output", filePath)
		assert.NoError(t, err)
		expectedSaveOutput := `Pulling image: docker.io/alpine/curl:8.9.1
Pulling image: docker.io/busybox:1.36.1
Tarball created successfully: image-load.tgz`
		assert.Equal(t, expectedSaveOutput, output)

		// Check if the file exists
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			t.Errorf("File was not created: %s", filePath)
		}

		output, err = executeCommand(rootCmd, "load", filePath, "-r", "ttl.sh")
		assert.NoError(t, err)
		expectedLoadOutput := `Successfully pushed image ttl.sh/curl:8.9.1
Successfully pushed image ttl.sh/busybox:1.36.1`
		assert.Equal(t, expectedLoadOutput, output)

		// Clean up: remove the file after the test
		err = os.Remove(filePath)
		if err != nil {
			t.Fatalf("Failed to remove file: %v", err)
		}
	})

}
