package crdchart

import (
	"fmt"

	"helm.sh/helm/v3/pkg/chart/loader"
)

// SourceMeta is the identity of the source chart, used to stamp the
// generated datarobot-infra chart for traceability.
type SourceMeta struct {
	Name       string
	Version    string
	AppVersion string
}

// ReadSourceChartMeta loads a chart (directory or .tgz) and returns its
// Chart.yaml identity fields verbatim.
func ReadSourceChartMeta(chartPath string) (SourceMeta, error) {
	c, err := loader.Load(chartPath)
	if err != nil {
		return SourceMeta{}, fmt.Errorf("error loading chart %s: %w", chartPath, err)
	}
	if c.Metadata == nil {
		return SourceMeta{}, fmt.Errorf("chart %s has no metadata", chartPath)
	}
	return SourceMeta{
		Name:       c.Metadata.Name,
		Version:    c.Metadata.Version,
		AppVersion: c.Metadata.AppVersion,
	}, nil
}
