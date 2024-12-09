package cmd

import (
	"fmt"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/datarobot-oss/helm-datarobot-plugin/pkg/image_uri"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/engine"
)

var generateCmd = &cobra.Command{
	Use:          "generate",
	Short:        "generate",
	Aliases:      []string{"gen", "genera"},
	SilenceUsage: true,
	Long: strings.Replace(`

This command is designed to extract all images and generate the image document annotations from a given change

Example:
'''sh
$ helm datarobot generate chart.tgz

'''`, "'", "`", -1),
	Args: cobra.MinimumNArgs(1), // Requires at least one argument (file path)
	RunE: func(cmd *cobra.Command, args []string) error {
		chartPath := args[0]
		chart, err := loader.Load(chartPath)
		if err != nil {
			return fmt.Errorf("Error loading chart %s: %v", chartPath, err)
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

		var sb strings.Builder
		uniqueEntries := make(map[string]struct{})
		for _, template := range chart.Templates {
			fileName, _ := template.Name, template.Data
			// We only apply the following lint rules to yaml files
			if filepath.Ext(fileName) != ".yaml" || filepath.Ext(fileName) == ".yml" {
				continue
			}

			renderedContent := renderedContentMap[path.Join(chart.Name(), fileName)]
			if generateDebug {
				fmt.Printf("---\n# Source: %s\n%s\n", fileName, renderedContent)
			}

			manifestImages, err := ExtractImagesFromManifest(renderedContent)
			if err != nil {
				return fmt.Errorf("Error ExtractImagesFromManifest chart %s: %v", chartPath, err)
			}

			re := regexp.MustCompile("[^a-zA-Z0-9]+")

			for _, item := range manifestImages {
				iUri, err := image_uri.NewDockerUri(item)
				if err != nil {
					return err
				}
				uniqueKey := iUri.ImageName + "_" + re.ReplaceAllString(iUri.Tag, "")
				// Check if the item is already in the map
				if _, exists := uniqueEntries[uniqueKey]; !exists {
					// If not, add it to the map and the finalSlice
					uniqueEntries[uniqueKey] = struct{}{}
					sb.WriteString(fmt.Sprintf("- name: %s\n", uniqueKey))
					sb.WriteString(fmt.Sprintf("  image: %s\n", item))
				}
			}

		}

		output := map[string]interface{}{
			"annotations": map[string]string{
				string(annotation): sb.String(),
			},
		}

		yamlData, err := yaml.Marshal(output)
		if err != nil {
			return fmt.Errorf("Error converting to YAML: %v\n", err)
		}

		// Print the YAML output
		cmd.Println(string(yamlData))

		return nil
	},
}

var generateDebug bool

func init() {
	rootCmd.AddCommand(generateCmd)
	generateCmd.Flags().StringVarP(&annotation, "annotation", "a", "datarobot.com/images", "annotation to lookup")
	generateCmd.Flags().BoolVarP(&generateDebug, "debug", "d", false, "debug")
}
