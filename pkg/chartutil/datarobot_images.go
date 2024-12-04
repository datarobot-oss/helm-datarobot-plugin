package chartutil

import (
	"bytes"
	"text/template"

	"gopkg.in/yaml.v2"
	"helm.sh/helm/v3/pkg/chart"
)

type DatarobotImageDeclaration struct {
	Name  string `yaml:"name"`
	Image string `yaml:"image"`
	Tag   string `yaml:"tag,omitempty"`
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
