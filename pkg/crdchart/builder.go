package crdchart

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chartutil"
)

// BuildChart assembles the standalone datarobot-infra chart in memory:
// one templates/crds.yaml holding every (already transformed) CRD body.
func BuildChart(crds []Resource, src SourceMeta) (*chart.Chart, error) {
	bodies := make([]string, 0, len(crds))
	for _, r := range crds {
		bodies = append(bodies, strings.TrimSpace(r.RawYAML))
	}
	crdsYAML := strings.Join(bodies, "\n---\n") + "\n"

	c := &chart.Chart{
		Metadata: &chart.Metadata{
			APIVersion:  "v2",
			Name:        "datarobot-infra",
			Type:        "application",
			Version:     src.Version,
			AppVersion:  src.AppVersion,
			Description: fmt.Sprintf("DataRobot CRDs extracted from %s %s", src.Name, src.Version),
		},
		Templates: []*chart.File{
			{Name: "templates/crds.yaml", Data: []byte(crdsYAML)},
		},
	}
	return c, nil
}

// PackageChart saves c as a .tgz at outPath. chartutil.Save writes into a
// directory under its own naming; we then move to outPath with a
// copy-fallback so cross-filesystem destinations work.
func PackageChart(c *chart.Chart, outPath string) error {
	tmpDir, err := os.MkdirTemp("", "crdchart-pkg-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	saved, err := chartutil.Save(c, tmpDir)
	if err != nil {
		return fmt.Errorf("failed to package chart: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
		return err
	}
	if err := moveFile(saved, outPath); err != nil {
		return fmt.Errorf("failed to write %s: %w", outPath, err)
	}
	return nil
}

// moveFile renames src to dst, falling back to copy+remove when the two
// live on different filesystems (os.Rename returns a cross-device error).
func moveFile(src, dst string) error {
	if err := os.Rename(src, dst); err == nil {
		return nil
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	if err := out.Sync(); err != nil {
		return err
	}
	return os.Remove(src)
}
