package cmd

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCommandSyncLive(t *testing.T) {
	t.Run("test-chart4 ttl.sh", func(t *testing.T) {
		output, err := executeCommand(rootCmd, "sync ../tests/charts/test-chart4 -r ttl.sh --overwrite")
		assert.NoError(t, err)
		// Expected output to compare
		expectedOutput := `Pulling image: docker.io/alpine/curl:8.9.1
Pushing image: ttl.sh/alpine/curl:stable

Pulling image: docker.io/busybox:1.36.1
Pushing image: ttl.sh/busybox:simple

Pulling image: docker.io/alpine/curl:8.10.0
Pushing image: ttl.sh/alpine/curl:8.10.0`
		assert.Equal(t, expectedOutput, output)
	})
	// 	t.Run("test-chart4 ttl.sh with proxy", func(t *testing.T) {
	// 		// Capture the output
	// 		var stdoutBuf bytes.Buffer
	// 		rootCmd.SetOut(io.Writer(&stdoutBuf))

	// 		_ = os.Setenv("HTTP_PROXY", "http:/47.89.184.18:3128")
	// 		_ = os.Setenv("HTTPS_PROXY", "http:/47.89.184.18:3128")

	// 		// Set arguments for the command (simulate CLI input)
	// 		arg := "sync ../tests/charts/test-chart4 -r ttl.sh --dry-run=false"
	// 		rootCmd.SetArgs(strings.Fields(arg))

	// 		// Execute command while capturing output
	// 		err := rootCmd.Execute()
	// 		assert.NoError(t, err)

	// 		// Expected output to compare
	// 		expectedOutput := `Pulling image: docker.io/alpine/curl:8.9.1
	// Pushing image: ttl.sh/alpine/curl:stable

	// Pulling image: docker.io/busybox:1.36.1
	// Pushing image: ttl.sh/busybox:simple

	// Pulling image: docker.io/alpine/curl:8.10.0
	// Pushing image: ttl.sh/alpine/curl:8.10.0

	// `
	//
	//		// Compare the actual output with the expected output
	//		assert.Equal(t, expectedOutput, stdoutBuf.String())
	//	})
}

func TestCommandSync(t *testing.T) {
	t.Run("test-chart4", func(t *testing.T) {
		output, err := executeCommand(rootCmd, "sync ../tests/charts/test-chart4 -r registry.example.com -u testuser -p testpass --dry-run")
		assert.NoError(t, err)
		expectedOutput := `[Dry-Run] Pulling image: docker.io/alpine/curl:8.9.1
[Dry-Run] Pushing image: registry.example.com/alpine/curl:stable

[Dry-Run] Pulling image: docker.io/busybox:1.36.1
[Dry-Run] Pushing image: registry.example.com/busybox:simple

[Dry-Run] Pulling image: docker.io/alpine/curl:8.10.0
[Dry-Run] Pushing image: registry.example.com/alpine/curl:8.10.0`
		assert.Equal(t, expectedOutput, output)
	})

	t.Run("prefix-suffix", func(t *testing.T) {
		output, err := executeCommand(rootCmd, "sync ../tests/charts/test-chart4 -r registry.example.com --dry-run -a custom/images --prefix prefix --suffix suffix ")
		assert.NoError(t, err)
		expectedOutput := `[Dry-Run] Pulling image: docker.io/datarobotdev/test-image4:4.0.0
[Dry-Run] Pushing image: registry.example.com/prefix/datarobotdev/suffix/test-image4:4.0.0`

		assert.Equal(t, expectedOutput, output)
	})

	t.Run("set-repo", func(t *testing.T) {
		output, err := executeCommand(rootCmd, "sync ../tests/charts/test-chart4 -r ocp.example.com --dry-run -a custom/images --repo openshift-image-registry/test ")
		assert.NoError(t, err)
		expectedOutput := `[Dry-Run] Pulling image: docker.io/datarobotdev/test-image4:4.0.0
[Dry-Run] Pushing image: ocp.example.com/openshift-image-registry/test/test-image4:4.0.0`

		assert.Equal(t, expectedOutput, output)
	})

	t.Run("skip-image-group", func(t *testing.T) {
		output, err := executeCommand(rootCmd, "sync ../tests/charts/test-chart6 -r registry.example.com --dry-run -a image/groups --skip-group test1 --skip-group test2")
		assert.NoError(t, err)
		expectedOutput := `Skipping image: docker.io/alpine/curl:8.9.10

Skipping image: docker.io/alpine/curl:8.9.11

Skipping image: docker.io/alpine/curl:8.9.2

[Dry-Run] Pulling image: docker.io/alpine/curl:8.9.3
[Dry-Run] Pushing image: registry.example.com/alpine/curl:8.9.3`

		assert.Equal(t, expectedOutput, output)
	})
	t.Run("skip-image", func(t *testing.T) {
		output, err := executeCommand(rootCmd, "sync ../tests/charts/test-chart6 -r registry.example.com --dry-run -a image/groups --skip-image docker.io/alpine/curl:8.9.10 --skip-image docker.io/alpine/curl:8.9.11 --skip-image docker.io/alpine/curl:8.9.2")
		assert.NoError(t, err)
		expectedOutput := `Skipping image: docker.io/alpine/curl:8.9.10

Skipping image: docker.io/alpine/curl:8.9.11

Skipping image: docker.io/alpine/curl:8.9.2

[Dry-Run] Pulling image: docker.io/alpine/curl:8.9.3
[Dry-Run] Pushing image: registry.example.com/alpine/curl:8.9.3`

		assert.Equal(t, expectedOutput, output)
	})

	t.Run("local-registry-insecure", func(t *testing.T) {
		os.Setenv("REGISTRY_USERNAME", "admin")
		os.Setenv("REGISTRY_PASSWORD", "pass")
		os.Setenv("REGISTRY_HOST", "localhost:5000")
		os.Setenv("SKIP_TLS_VERIFY", "true")
		output, err := executeCommand(rootCmd, "sync ../tests/charts/test-chart6 -a ex4")
		assert.NoError(t, err)
		expectedLoadOutput := `Pulling image: docker.io/alpine/curl:8.11.1
Pushing image: localhost:5000/alpine/curl:8.11.1`
		assert.Equal(t, expectedLoadOutput, output)
	})
	t.Run("duplicated", func(t *testing.T) {
		os.Setenv("REGISTRY_USERNAME", "admin")
		os.Setenv("REGISTRY_PASSWORD", "pass")
		os.Setenv("REGISTRY_HOST", "localhost:5000")
		os.Setenv("SKIP_TLS_VERIFY", "true")

		output, err := executeCommand(rootCmd, "sync ../tests/charts/test-chart6 -a ex4")
		assert.NoError(t, err)
		expectedLoadOutput := `image localhost:5000/alpine/curl:8.11.1 already exists in the registry`
		assert.Equal(t, expectedLoadOutput, output)
	})
	t.Run("overwrite", func(t *testing.T) {
		os.Setenv("REGISTRY_USERNAME", "admin")
		os.Setenv("REGISTRY_PASSWORD", "pass")
		os.Setenv("REGISTRY_HOST", "localhost:5000")
		os.Setenv("SKIP_TLS_VERIFY", "true")

		output, err := executeCommand(rootCmd, "sync ../tests/charts/test-chart6 --overwrite -a ex4")
		assert.NoError(t, err)
		expectedLoadOutput := `Pulling image: docker.io/alpine/curl:8.11.1
Pushing image: localhost:5000/alpine/curl:8.11.1`
		assert.Equal(t, expectedLoadOutput, output)
	})
}
