package cmd

import (
	"crypto/sha256"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Publish-readiness: identical inputs -> byte-identical tarball (spec §10).
//
// Note: this compares raw tarball bytes (sha256). In principle gzip/tar
// mtime headers written by chartutil.Save could make this flaky even when
// the chart content itself is deterministic; if that is ever observed,
// switch to a content-level comparison (loader.Load both tgzs, sort
// templates by name, compare Data bytes) instead of raw bytes. As of this
// writing, raw-bytes comparison is stable across repeated runs.
func TestInfraChartDeterministic(t *testing.T) {
	tmp := t.TempDir()
	outA := filepath.Join(tmp, "a.tgz")
	outB := filepath.Join(tmp, "b.tgz")

	_, err := executeCommand(rootCmd, "infra-chart ../tests/charts/crd-test-chart -o "+outA+" --pipeline-sa pipeline --namespace dr")
	assert.NoError(t, err)
	_, err = executeCommand(rootCmd, "infra-chart ../tests/charts/crd-test-chart -o "+outB+" --pipeline-sa pipeline --namespace dr")
	assert.NoError(t, err)

	a, err := os.ReadFile(outA)
	assert.NoError(t, err)
	b, err := os.ReadFile(outB)
	assert.NoError(t, err)
	assert.Equal(t, sha256.Sum256(a), sha256.Sum256(b), "generate must be byte-identical for identical inputs")
}
