package adminchart

import (
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
)

// ReadSourceChartMeta loads a chart from chartPath and returns its metadata.
func ReadSourceChartMeta(chartPath string) (*chart.Metadata, error) {
	c, err := loader.Load(chartPath)
	if err != nil {
		return nil, err
	}
	return c.Metadata, nil
}
