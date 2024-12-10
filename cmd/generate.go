package cmd

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/datarobot-oss/helm-datarobot-plugin/pkg/image_uri"
	"github.com/datarobot-oss/helm-datarobot-plugin/pkg/render_helper"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
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
		manifest, err := render_helper.NewRenderItems(chartPath)
		if err != nil {
			return fmt.Errorf("Error loading chart %s: %v", chartPath, err)
		}

		uniqueEntries := make(map[string]string)
		for fileName, template := range manifest {
			// // We only apply the following lint rules to yaml files
			if filepath.Ext(fileName) != ".yaml" || filepath.Ext(fileName) == ".yml" {
				continue
			}

			if generateDebug {
				fmt.Printf("---\n# Source: %s\n%s\n", fileName, template)
			}

			manifestImages, err := ExtractImagesFromManifest(template)
			if err != nil {
				return fmt.Errorf("Error ExtractImagesFromManifest chart %s: %v", fileName, err)
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
					uniqueEntries[uniqueKey] = item
				}
			}

		}

		var sb strings.Builder
		uniqueEntries = render_helper.SortMap(uniqueEntries)
		for key, value := range uniqueEntries {
			sb.WriteString(fmt.Sprintf("- name: %s\n", key))
			sb.WriteString(fmt.Sprintf("  image: %s\n", value))
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
