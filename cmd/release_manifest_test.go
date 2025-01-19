package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCommandReleaseManifest(t *testing.T) {
	t.Run("Test test-chart1", func(t *testing.T) {
		output, err := executeCommand(rootCmd, "release-manifest ../testdata/test-chart1 -a \"datarobot.com/images\"")
		assert.NoError(t, err)
		// Expected output to compare
		expectedOutput := `images:
  test-image1.tar.zst:
    source: docker.io/datarobotdev/test-image1:1.0.0
    name: docker.io/datarobotdev/test-image1
    tag: 1.0.0
  test-image2.tar.zst:
    source: docker.io/datarobotdev/test-image2:2.0.0
    name: docker.io/datarobotdev/test-image2
    tag: 2.0.0
  test-image3.tar.zst:
    source: docker.io/datarobotdev/test-image3:3.0.0
    name: docker.io/datarobotdev/test-image3
    tag: 3.0.0`
		assert.Equal(t, expectedOutput, output)
	})

	t.Run("Test test-chart4", func(t *testing.T) {
		output, err := executeCommand(rootCmd, "release-manifest ../testdata/test-chart4 -a \"custom/images\"")
		assert.NoError(t, err)
		expectedOutput := `images:
  test-image4.tar.zst:
    source: docker.io/datarobotdev/test-image4:4.0.0
    name: docker.io/datarobotdev/test-image4
    tag: 4.0.0`
		assert.Equal(t, expectedOutput, output)
	})

	t.Run("Test test-chart4-datarobot", func(t *testing.T) {
		output, err := executeCommand(rootCmd, "release-manifest ../testdata/test-chart4 -a \"datarobot.com/images\"")
		assert.NoError(t, err)
		expectedOutput := `images:
  test-image3.tar.zst:
    source: docker.io/alpine/curl:8.9.1
    name: docker.io/alpine/curl
    tag: stable
  test-image30.tar.zst:
    source: docker.io/busybox:1.36.1
    name: docker.io/busybox
    tag: simple
  test-image31.tar.zst:
    source: docker.io/alpine/curl:8.10.0
    name: docker.io/alpine/curl
    tag: 8.10.0`

		assert.Equal(t, expectedOutput, output)
	})
	t.Run("Test test-chart4-duplicated", func(t *testing.T) {
		output, err := executeCommand(rootCmd, "release-manifest ../testdata/test-chart4 -a \"custom/images-duplicated\"")
		assert.NoError(t, err)
		expectedOutput := `images:
  test-image4.tar.zst:
    source: docker.io/datarobotdev/test-image4:4.0.0
    name: docker.io/datarobotdev/test-image4
    tag: 4.0.0`
		assert.Equal(t, expectedOutput, output)
	})
	t.Run("selected-labels", func(t *testing.T) {
		output, err := executeCommand(rootCmd, "release-manifest ../testdata/test-chart6  -a bitnami -l org.opencontainers.image.title -l org.opencontainers.image.base.name ")
		assert.NoError(t, err)
		expectedOutput := `images:
  redis.tar.zst:
    source: docker.io/bitnami/redis:7.4.2-debian-12-r0
    name: docker.io/bitnami/redis
    tag: 7.4.2-debian-12-r0
    labels:
      org.opencontainers.image.base.name: docker.io/bitnami/minideb:bookworm
      org.opencontainers.image.title: redis`
		assert.Equal(t, expectedOutput, output)
	})
	t.Run("all-labels", func(t *testing.T) {
		output, err := executeCommand(rootCmd, "release-manifest ../testdata/test-chart6  -a bitnami --all-labels ")
		assert.NoError(t, err)
		expectedOutput := `images:
  redis.tar.zst:
    source: docker.io/bitnami/redis:7.4.2-debian-12-r0
    name: docker.io/bitnami/redis
    tag: 7.4.2-debian-12-r0
    labels:
      com.vmware.cp.artifact.flavor: sha256:c50c90cfd9d12b445b011e6ad529f1ad3daea45c26d20b00732fae3cd71f6a83
      org.opencontainers.image.base.name: docker.io/bitnami/minideb:bookworm
      org.opencontainers.image.created: "2025-01-07T17:15:12Z"
      org.opencontainers.image.description: Application packaged by Broadcom, Inc.
      org.opencontainers.image.documentation: https://github.com/bitnami/containers/tree/main/bitnami/redis/README.md
      org.opencontainers.image.licenses: Apache-2.0
      org.opencontainers.image.ref.name: 7.4.2-debian-12-r0
      org.opencontainers.image.source: https://github.com/bitnami/containers/tree/main/bitnami/redis
      org.opencontainers.image.title: redis
      org.opencontainers.image.vendor: Broadcom, Inc.
      org.opencontainers.image.version: 7.4.2`
		assert.Equal(t, expectedOutput, output)
	})
}
