package crdchart

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"helm.sh/helm/v3/pkg/chart/loader"
)

func TestBuildChart(t *testing.T) {
	crds := []Resource{
		{Kind: "CustomResourceDefinition", Name: "a", RawYAML: "kind: CustomResourceDefinition\nmetadata:\n  name: a"},
		{Kind: "CustomResourceDefinition", Name: "b", RawYAML: "kind: CustomResourceDefinition\nmetadata:\n  name: b"},
	}
	c, err := BuildChart(crds, SourceMeta{Name: "datarobot-prime", Version: "11.10.88", AppVersion: "11.10.88"})
	assert.NoError(t, err)
	assert.Equal(t, "datarobot-crds", c.Metadata.Name)
	assert.Equal(t, "v2", c.Metadata.APIVersion)
	assert.Equal(t, "11.10.88", c.Metadata.Version)
	assert.Len(t, c.Templates, 1)
	assert.Equal(t, "templates/crds.yaml", c.Templates[0].Name)
	body := string(c.Templates[0].Data)
	assert.Contains(t, body, "name: a")
	assert.Contains(t, body, "name: b")
	assert.Contains(t, body, "\n---\n")
}

func TestPackageChartWritesTgz(t *testing.T) {
	c, err := BuildChart(
		[]Resource{{Name: "a", RawYAML: "kind: CustomResourceDefinition\nmetadata:\n  name: a"}},
		SourceMeta{Name: "src", Version: "1.0.0", AppVersion: "1.0.0"},
	)
	assert.NoError(t, err)

	out := filepath.Join(t.TempDir(), "datarobot-crds-1.0.0.tgz")
	assert.NoError(t, PackageChart(c, out))

	fi, err := os.Stat(out)
	assert.NoError(t, err)
	assert.True(t, fi.Size() > 0)

	// The written tarball must load back as a valid chart.
	reloaded, err := loader.Load(out)
	assert.NoError(t, err)
	assert.Equal(t, "datarobot-crds", reloaded.Metadata.Name)
}
