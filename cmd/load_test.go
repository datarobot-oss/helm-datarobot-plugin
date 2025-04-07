package cmd

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

const LOAD_TEST_ARCHIVE = "image-load.tar.zst"

func TestCommandLoad(t *testing.T) {
	t.Run("gen-tarball", func(t *testing.T) {

		output, err := executeCommand(rootCmd, "save ../tests/charts/test-chart4 -a custom/loadimages --output "+LOAD_TEST_ARCHIVE)
		assert.NoError(t, err)
		expectedSaveOutput := `Pulling image: docker.io/alpine/curl:8.9.1
Pulling image: docker.io/busybox:1.36.1
Tarball created successfully: image-load.tar.zst`
		assert.Equal(t, expectedSaveOutput, output)

		// Check if the file exists
		if _, err := os.Stat(LOAD_TEST_ARCHIVE); os.IsNotExist(err) {
			t.Errorf("File was not created: %s", LOAD_TEST_ARCHIVE)
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
	t.Run("env-var", func(t *testing.T) {
		os.Setenv("REGISTRY_HOST", "localhost:5000")
		output, err := executeCommand(rootCmd, "load "+LOAD_TEST_ARCHIVE+" --dry-run ")
		assert.NoError(t, err)
		expectedLoadOutput := `[Dry-Run] Pushing image: localhost:5000/alpine/curl:8.9.1
[Dry-Run] Pushing image: localhost:5000/busybox:1.36.1`
		assert.Equal(t, expectedLoadOutput, output)
	})
	t.Run("local-registry-insecure", func(t *testing.T) {
		os.Setenv("REGISTRY_USERNAME", "admin")
		os.Setenv("REGISTRY_PASSWORD", "pass")

		output, err := executeCommand(rootCmd, "load "+LOAD_TEST_ARCHIVE+" -r localhost:5000 --insecure")
		assert.NoError(t, err)
		expectedLoadOutput := `Successfully pushed image localhost:5000/alpine/curl:8.9.1
Successfully pushed image localhost:5000/busybox:1.36.1`
		assert.Equal(t, expectedLoadOutput, output)
	})

	t.Run("local-registry-ca", func(t *testing.T) {
		os.Setenv("REGISTRY_USERNAME", "admin")
		os.Setenv("REGISTRY_PASSWORD", "pass")
		output, err := executeCommand(rootCmd, "load "+LOAD_TEST_ARCHIVE+" -r localhost:5000 --ca-cert ../tests/registry/certs/ca.crt")
		assert.NoError(t, err)
		expectedLoadOutput := `Successfully pushed image localhost:5000/alpine/curl:8.9.1
Successfully pushed image localhost:5000/busybox:1.36.1`
		assert.Equal(t, expectedLoadOutput, output)
	})

	t.Run("prefix-suffix", func(t *testing.T) {
		output, err := executeCommand(rootCmd, "load "+LOAD_TEST_ARCHIVE+" -r ttl.sh --prefix prefix --suffix suffix")
		assert.NoError(t, err)
		expectedLoadOutput := `Successfully pushed image ttl.sh/prefix/alpine/suffix/curl:8.9.1
Successfully pushed image ttl.sh/prefix/suffix/busybox:1.36.1`
		assert.Equal(t, expectedLoadOutput, output)
	})

	t.Run("duplicated", func(t *testing.T) {
		output, err := executeCommand(rootCmd, "load "+LOAD_TEST_ARCHIVE+" -r localhost:5000 -u admin -p pass --insecure")
		assert.NoError(t, err)
		expectedLoadOutput := `image localhost:5000/alpine/curl:8.9.1 already exists in the registry
Successfully pushed image localhost:5000/alpine/curl:8.9.1
image localhost:5000/busybox:1.36.1 already exists in the registry
Successfully pushed image localhost:5000/busybox:1.36.1`
		assert.Equal(t, expectedLoadOutput, output)
	})
	t.Run("overwrite", func(t *testing.T) {
		output, err := executeCommand(rootCmd, "load "+LOAD_TEST_ARCHIVE+" -r localhost:5000 -u admin -p pass --overwrite --insecure")
		assert.NoError(t, err)
		expectedLoadOutput := `Successfully pushed image localhost:5000/alpine/curl:8.9.1
Successfully pushed image localhost:5000/busybox:1.36.1`
		assert.Equal(t, expectedLoadOutput, output)
	})

	t.Run("repo", func(t *testing.T) {
		output, err := executeCommand(rootCmd, "load "+LOAD_TEST_ARCHIVE+" -r ocp.example.com --dry-run --repo openshift-image-registry/test ")
		assert.NoError(t, err)
		expectedLoadOutput := `[Dry-Run] Pushing image: ocp.example.com/openshift-image-registry/test/curl:8.9.1
[Dry-Run] Pushing image: ocp.example.com/openshift-image-registry/test/busybox:1.36.1`
		assert.Equal(t, expectedLoadOutput, output)
	})
	t.Run("cleanup-tarball", func(t *testing.T) {
		// Clean up: remove the file after the test
		err := os.Remove(LOAD_TEST_ARCHIVE)
		if err != nil {
			t.Fatalf("Failed to remove file: %v", err)
		}
	})

}
