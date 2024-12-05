package cmd

import (
	"fmt"
	"path"
	"path/filepath"
	"strings"

	dr_chartutil "github.com/datarobot-oss/helm-datarobot-plugin/pkg/chartutil"
	"github.com/spf13/cobra"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/engine"
	v1 "k8s.io/api/apps/v1"
	batch_v1 "k8s.io/api/batch/v1"
	"sigs.k8s.io/yaml"
)

// isImageAllowed checks if the image is in the allowed list
func isImageAllowed(image string, imageDoc []dr_chartutil.DatarobotImageDeclaration) bool {
	for _, im := range imageDoc {
		if strings.TrimSpace(image) == strings.TrimSpace(im.Image) {
			return true
		}
	}
	return false
}

func ExtractImagesFromManifest(manifest string) ([]string, error) {
	var manifestImages []string

	var deployment v1.Deployment
	if err := yaml.Unmarshal([]byte(manifest), &deployment); err != nil {
		return nil, fmt.Errorf("Error unmarshalling YAML: %v\n", err)
	}
	// Collect images from init containers
	for _, initContainer := range deployment.Spec.Template.Spec.InitContainers {
		manifestImages = append(manifestImages, initContainer.Image)
	}

	// Collect images from regular containers
	for _, container := range deployment.Spec.Template.Spec.Containers {
		manifestImages = append(manifestImages, container.Image)
	}

	var statefulSet v1.StatefulSet
	if err := yaml.Unmarshal([]byte(manifest), &statefulSet); err != nil {
		return nil, fmt.Errorf("Error unmarshalling YAML: %v\n", err)
	}
	// Collect images from init containers
	for _, initContainer := range statefulSet.Spec.Template.Spec.InitContainers {
		manifestImages = append(manifestImages, initContainer.Image)
	}

	// Collect images from regular containers
	for _, container := range statefulSet.Spec.Template.Spec.Containers {
		manifestImages = append(manifestImages, container.Image)
	}

	var job batch_v1.Job
	if err := yaml.Unmarshal([]byte(manifest), &job); err != nil {
		return nil, fmt.Errorf("Error unmarshalling YAML: %v\n", err)
	}
	// Collect images from regular containers
	for _, container := range job.Spec.Template.Spec.Containers {
		manifestImages = append(manifestImages, container.Image)
	}

	var cronJob batch_v1.CronJob
	if err := yaml.Unmarshal([]byte(manifest), &cronJob); err != nil {
		return nil, fmt.Errorf("Error unmarshalling YAML: %v\n", err)
	}
	// Collect images from regular containers
	for _, container := range cronJob.Spec.JobTemplate.Spec.Template.Spec.Containers {
		manifestImages = append(manifestImages, container.Image)
	}
	return manifestImages, nil
}

var validateCmd = &cobra.Command{
	Use:          "validate",
	Short:        "validate",
	Aliases:      []string{"valid", "check"},
	SilenceUsage: true,
	Long: strings.Replace(`

This command is designed to validate all images presnet in a chart are declared inside the annotation

Example:
'''sh
$ helm datarobot validate chart.tgz

'''`, "'", "`", -1),
	Args: cobra.MinimumNArgs(1), // Requires at least one argument (file path)
	RunE: func(cmd *cobra.Command, args []string) error {
		chartPath := args[0]
		chart, err := loader.Load(chartPath)
		if err != nil {
			return fmt.Errorf("Error loading chart %s: %v", chartPath, err)
		}

		imageDoc, err := ExtractImagesFromCharts(args)
		if err != nil {
			return fmt.Errorf("Error ExtractImagesFromCharts: %v", err)
		}

		options := chartutil.ReleaseOptions{
			Name:      "test-release",
			Namespace: "test",
		}
		values := map[string]interface{}{}
		caps := chartutil.DefaultCapabilities.Copy()

		cvals, err := chartutil.CoalesceValues(chart, values)
		if err != nil {
			return fmt.Errorf("Error CoalesceValues chart %s: %v", chartPath, err)
		}

		valuesToRender, err := chartutil.ToRenderValuesWithSchemaValidation(chart, cvals, options, caps, true)
		if err != nil {
			return fmt.Errorf("Error ToRenderValuesWithSchemaValidation chart %s: %v", chartPath, err)
		}
		var e engine.Engine

		renderedContentMap, err := e.Render(chart, valuesToRender)
		if err != nil {
			return fmt.Errorf("Error Render chart %s: %v", chartPath, err)
		}
		for _, template := range chart.Templates {
			fileName, _ := template.Name, template.Data
			// We only apply the following lint rules to yaml files
			if filepath.Ext(fileName) != ".yaml" || filepath.Ext(fileName) == ".yml" {
				continue
			}

			renderedContent := renderedContentMap[path.Join(chart.Name(), fileName)]
			if validateDebug {
				fmt.Printf("---\n# Source: %s\n%s\n", fileName, renderedContent)
			}

			manifestImages, err := ExtractImagesFromManifest(renderedContent)
			if err != nil {
				return fmt.Errorf("Error ExtractImagesFromManifest chart %s: %v", chartPath, err)
			}
			// Validate manifestImages against the imageDoc
			for _, image := range manifestImages {
				// fmt.Print(imageDoc)
				if !isImageAllowed(image, imageDoc) {
					return fmt.Errorf("Image not defined in as imageDoc: %s\n", image)
				}
			}

		}

		cmd.Print("Image Doc Valid")

		return nil
	},
}

var validateDebug bool

func init() {
	rootCmd.AddCommand(validateCmd)
	validateCmd.Flags().StringVarP(&annotation, "annotation", "a", "datarobot.com/images", "annotation to lookup")
	validateCmd.Flags().BoolVarP(&validateDebug, "debug", "d", false, "debug")
}
