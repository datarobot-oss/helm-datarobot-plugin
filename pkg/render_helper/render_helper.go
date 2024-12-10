package render_helper

import (
	"fmt"
	"maps"
	"sort"

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

	return SortMap(renderItems), nil
}

// SortMap takes a map[string]string and returns a new map[string]string
// with the key-value pairs sorted by keys.
func SortMap(m map[string]string) map[string]string {
	// Step 1: Extract keys
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}

	// Step 2: Sort keys
	sort.Strings(keys)

	// Step 3: Create a new map to hold the sorted key-value pairs
	sortedMap := make(map[string]string)
	for _, key := range keys {
		sortedMap[key] = m[key]
	}

	return sortedMap
}
