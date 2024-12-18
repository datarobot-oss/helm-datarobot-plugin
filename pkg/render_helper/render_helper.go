package render_helper

import (
	"fmt"
	"os"
	"path"

	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/engine"
)

func NewRenderItems(chartPath string, valueFiles []string, setValues []string) (map[string]string, error) {
	loadedChart, err := loader.Load(chartPath)
	if err != nil {
		return nil, fmt.Errorf("Error loading chart %s: %v", chartPath, err)
	}

	subChartsPath := path.Join(chartPath, "charts")
	if fi, err := os.Stat(subChartsPath); err == nil && fi.IsDir() {
		subCharts, err := os.ReadDir(subChartsPath)
		if err == nil && len(subCharts) > 0 {
			for _, subChart := range subCharts {
				subChartPath := path.Join(subChartsPath, subChart.Name())
				if !subChart.IsDir() || !isChartDirectory(subChartPath) {
					continue
				}
				loadedSubChart, err := loader.Load(path.Join(subChartsPath, subChart.Name()))
				if err != nil {
					return nil, fmt.Errorf("Error subChartsPath %s: %v", subChartPath, err)
				}
				loadedChart.Values[subChart.Name()] = loadedSubChart.Values
			}
		}
	}

	options := chartutil.ReleaseOptions{
		Name:      "test-release",
		Namespace: "test",
	}

	values, err := loadValues(valueFiles, setValues)
	if err != nil {
		return nil, err
	}

	caps := chartutil.DefaultCapabilities.Copy()
	cvals, err := chartutil.CoalesceValues(loadedChart, values)
	if err != nil {
		return nil, fmt.Errorf("Error CoalesceValues chart %s: %v", chartPath, err)
	}

	valuesToRender, err := chartutil.ToRenderValuesWithSchemaValidation(loadedChart, cvals, options, caps, true)
	if err != nil {
		return nil, fmt.Errorf("Error ToRenderValuesWithSchemaValidation chart %s: %v", chartPath, err)
	}

	renderedContentMap, err := engine.Render(loadedChart, valuesToRender)
	if err != nil {
		return nil, fmt.Errorf("Error Render chart %s: %v", chartPath, err)
	}
	return renderedContentMap, nil
}
