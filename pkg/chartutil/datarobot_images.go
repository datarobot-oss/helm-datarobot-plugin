package chartutil

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"text/template"

	"gopkg.in/yaml.v2"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
)

type DatarobotImageDeclaration struct {
	Name  string `yaml:"name"`
	Image string `yaml:"image"`
	Tag   string `yaml:"tag,omitempty"`
	Group string `yaml:"group,omitempty"`
}

type ChartImages struct {
	ChartFullPath string
	Images        []DatarobotImageDeclaration
	Err           error
}

func RenderDatarobotImages(c *chart.Chart, annotation string) *ChartImages {
	datarobotImages, exists := c.Metadata.Annotations[annotation]
	if !exists {
		return nil
	}

	result := ChartImages{
		ChartFullPath: c.ChartFullPath(),
		Images:        make([]DatarobotImageDeclaration, 0),
		Err:           nil,
	}

	chartMetaData := struct {
		chart.Metadata
		IsRoot bool
	}{*c.Metadata, c.IsRoot()}
	vals := map[string]interface{}{
		"Chart": chartMetaData,
	}
	tmpl := template.New("gotpl")
	tmpl.Option("missingkey=error")
	tmpl, err := tmpl.Parse(datarobotImages)
	if err != nil {
		result.Err = err
		return &result
	}
	var buffer bytes.Buffer
	err = tmpl.Execute(&buffer, vals)
	if err != nil {
		result.Err = err
		return &result
	}

	var imageDeclarations []DatarobotImageDeclaration
	err = yaml.Unmarshal(buffer.Bytes(), &imageDeclarations)
	if err != nil {
		result.Err = err
		return &result
	}
	result.Images = imageDeclarations
	return &result
}

func RecursiveRenderDatarobotImages(c *chart.Chart, annotation string) []ChartImages {
	result := make([]ChartImages, 0)

	chartImages := RenderDatarobotImages(c, annotation)
	if chartImages != nil {
		result = append(result, *chartImages)
	}

	for _, child := range c.Dependencies() {
		result = append(result, RecursiveRenderDatarobotImages(child, annotation)...)
	}

	return result
}

// ExtractImagesFromCharts loads all images from the given chart paths using the provided annotation.
func ExtractImagesFromCharts(args []string, annotation string) ([]DatarobotImageDeclaration, error) {
	allChartImages := make([]ChartImages, 0)
	for _, chartPath := range args {
		c, err := loader.Load(chartPath)
		if err != nil {
			return nil, fmt.Errorf("Error loading chart %s: %v", chartPath, err)
		}
		allChartImages = append(
			allChartImages,
			RecursiveRenderDatarobotImages(c, annotation)...,
		)
	}
	allImages := make([]DatarobotImageDeclaration, 0)
	allErrors := make([]string, 0)

	for _, ci := range allChartImages {
		if ci.Err != nil {
			formattedError := fmt.Sprintf("[%s] %s", ci.ChartFullPath, ci.Err.Error())
			allErrors = append(allErrors, formattedError)
		}
		allImages = append(allImages, ci.Images...)
	}
	if len(allErrors) > 0 {
		return nil, errors.New(strings.Join(allErrors, "\n"))
	}
	return allImages, nil
}
