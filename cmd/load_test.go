package cmd

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCommandLoad(t *testing.T) {
	t.Run("gen-tarball", func(t *testing.T) {
		filePath := "image-load.tgz"
		output, err := executeCommand(rootCmd, "save ../testdata/test-chart4 -a custom/loadimages --output "+filePath)
		assert.NoError(t, err)
		expectedSaveOutput := `Pulling image: docker.io/alpine/curl:8.9.1
Pulling image: docker.io/busybox:1.36.1
Tarball created successfully: image-load.tgz`
		assert.Equal(t, expectedSaveOutput, output)

		// Check if the file exists
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			t.Errorf("File was not created: %s", filePath)
		}
	})
	t.Run("check-registry-online", func(t *testing.T) {
		url := "https://localhost:5000/v2/_catalog/"
		username := "admin"
		password := "pass"

		err := checkRegistryOnline(url, username, password)
		if err != nil {
			t.Fatalf("Failed to check registry online: %v", err)
		}
	})
	t.Run("local-registry-insecure", func(t *testing.T) {
		os.Setenv("REGISTRY_USERNAME", "admin")
		os.Setenv("REGISTRY_PASSWORD", "pass")
		output, err := executeCommand(rootCmd, "load image-load.tgz -r localhost:5000 --insecure")
		assert.NoError(t, err)
		expectedLoadOutput := `Successfully pushed image localhost:5000/alpine/curl:8.9.1
Successfully pushed image localhost:5000/busybox:1.36.1`
		assert.Equal(t, expectedLoadOutput, output)
	})

	t.Run("local-registry-insecure", func(t *testing.T) {
		os.Setenv("REGISTRY_USERNAME", "admin")
		os.Setenv("REGISTRY_PASSWORD", "pass")
		output, err := executeCommand(rootCmd, "load image-load.tgz -r localhost:5000 --ca-cert ../tests/registry/certs/ca.crt")
		assert.NoError(t, err)
		expectedLoadOutput := `Successfully pushed image localhost:5000/alpine/curl:8.9.1
Successfully pushed image localhost:5000/busybox:1.36.1`
		assert.Equal(t, expectedLoadOutput, output)
	})

	t.Run("prefix-suffix", func(t *testing.T) {
		output, err := executeCommand(rootCmd, "load image-load.tgz -r ttl.sh --prefix prefix --suffix suffix")
		assert.NoError(t, err)
		expectedLoadOutput := `Successfully pushed image ttl.sh/prefix/alpine/suffix/curl:8.9.1
Successfully pushed image ttl.sh/prefix/suffix/busybox:1.36.1`
		assert.Equal(t, expectedLoadOutput, output)
	})

	t.Run("repo", func(t *testing.T) {
		output, err := executeCommand(rootCmd, "load image-load.tgz -r ocp.example.com --dry-run --repo openshift-image-registry/test ")
		assert.NoError(t, err)
		expectedLoadOutput := `[Dry-Run] Pushing image: ocp.example.com/openshift-image-registry/test/curl:8.9.1
[Dry-Run] Pushing image: ocp.example.com/openshift-image-registry/test/busybox:1.36.1`
		assert.Equal(t, expectedLoadOutput, output)
	})
	t.Run("cleanup-tarball", func(t *testing.T) {
		filePath := "image-load.tgz"
		// Clean up: remove the file after the test
		err := os.Remove(filePath)
		if err != nil {
			t.Fatalf("Failed to remove file: %v", err)
		}
	})

}
