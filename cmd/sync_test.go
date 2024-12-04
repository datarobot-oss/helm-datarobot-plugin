package cmd

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCommandSync(t *testing.T) {
	t.Run("test-chart4", func(t *testing.T) {
		// Capture the output
		var stdoutBuf bytes.Buffer
		rootCmd.SetOut(io.Writer(&stdoutBuf))

		// Set arguments for the command (simulate CLI input)
		arg := "sync ../testdata/test-chart4 -r registry.example.com -u testuser -p testpass --dry-run"
		rootCmd.SetArgs(strings.Fields(arg))

		// Execute command while capturing output
		err := rootCmd.Execute()
		assert.NoError(t, err)

		// Expected output to compare
		expectedOutput := `[Dry-Run] Pulling image: docker.io/alpine/curl:8.9.1
[Dry-Run] Pushing image: registry.example.com/alpine/curl:stable

[Dry-Run] Pulling image: docker.io/busybox:1.36.1
[Dry-Run] Pushing image: registry.example.com/busybox:simple

[Dry-Run] Pulling image: docker.io/alpine/curl:8.10.0
[Dry-Run] Pushing image: registry.example.com/alpine/curl:8.10.0

`
		// Compare the actual output with the expected output
		assert.Equal(t, expectedOutput, stdoutBuf.String())
	})
}
func TestCommandSyncLive(t *testing.T) {
	t.Run("test-chart4 ttl.sh", func(t *testing.T) {
		// Capture the output
		var stdoutBuf bytes.Buffer
		rootCmd.SetOut(io.Writer(&stdoutBuf))

		// Set arguments for the command (simulate CLI input)
		arg := "sync ../testdata/test-chart4 -r ttl.sh --dry-run=false"
		rootCmd.SetArgs(strings.Fields(arg))

		// Execute command while capturing output
		err := rootCmd.Execute()
		assert.NoError(t, err)

		// Expected output to compare
		expectedOutput := `Pulling image: docker.io/alpine/curl:8.9.1
Pushing image: ttl.sh/alpine/curl:stable

Pulling image: docker.io/busybox:1.36.1
Pushing image: ttl.sh/busybox:simple

Pulling image: docker.io/alpine/curl:8.10.0
Pushing image: ttl.sh/alpine/curl:8.10.0

`

		// Compare the actual output with the expected output
		assert.Equal(t, expectedOutput, stdoutBuf.String())
	})
	// 	t.Run("test-chart4 ttl.sh with proxy", func(t *testing.T) {
	// 		// Capture the output
	// 		var stdoutBuf bytes.Buffer
	// 		rootCmd.SetOut(io.Writer(&stdoutBuf))

	// 		_ = os.Setenv("HTTP_PROXY", "http:/47.89.184.18:3128")
	// 		_ = os.Setenv("HTTPS_PROXY", "http:/47.89.184.18:3128")

	// 		// Set arguments for the command (simulate CLI input)
	// 		arg := "sync ../testdata/test-chart4 -r ttl.sh --dry-run=false"
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
