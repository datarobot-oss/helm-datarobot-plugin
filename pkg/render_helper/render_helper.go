package render_helper

import (
	"fmt"
	"maps"

	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/engine"
)

func NewRenderItems(chartPath string) (map[string]string, error) {
	renderItems := make(map[string]string)
	chart, err := loader.Load(chartPath)
	if err != nil {
		return nil, fmt.Errorf("Error loading chart %s: %v", chartPath, err)
	}

	options := chartutil.ReleaseOptions{
		Name:      "test-release",
		Namespace: "test",
	}
	values := map[string]interface{}{}
	caps := chartutil.DefaultCapabilities.Copy()

	cvals, err := chartutil.CoalesceValues(chart, values)
	if err != nil {
		return nil, fmt.Errorf("Error CoalesceValues chart %s: %v", chartPath, err)
	}

	valuesToRender, err := chartutil.ToRenderValuesWithSchemaValidation(chart, cvals, options, caps, true)
	if err != nil {
		return nil, fmt.Errorf("Error ToRenderValuesWithSchemaValidation chart %s: %v", chartPath, err)
	}
	var e engine.Engine

	renderedContentMap, err := e.Render(chart, valuesToRender)
	if err != nil {
		return nil, fmt.Errorf("Error Render chart %s: %v", chartPath, err)
	}
	maps.Copy(renderItems, renderedContentMap)

	for _, child := range chart.Dependencies() {
		// fmt.Print(child)
		renderedContentMapChild, err := e.Render(child, valuesToRender)
		if err != nil {
			return nil, fmt.Errorf("Error Render chart %s: %v", chartPath, err)
		}

		maps.Copy(renderItems, renderedContentMapChild)
	}

	return renderItems, nil
}
