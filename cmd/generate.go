package cmd

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/datarobot-oss/helm-datarobot-plugin/pkg/chartutil"
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
		manifest, err := render_helper.RenderChart(chartPath, g.ValueFiles, g.Values)
		if err != nil {
			return fmt.Errorf("Error loading chart %s: %v", chartPath, err)
		}

		uniqueEntries := make(map[string]string)
		for _, template := range strings.Split(manifest, "\n---\n") {

			if g.Debug {
				fmt.Printf("---\n%s\n", template)
			}

			manifestImages, err := ExtractImagesFromManifest(template)
			if err != nil {
				return fmt.Errorf("Error ExtractImagesFromManifest chart: %v", err)
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

		var keys []string
		for key := range uniqueEntries {
			keys = append(keys, key)
		}

		// Sort the keys
		sort.Strings(keys)

		// Create a slice to hold the items
		var items []chartutil.DatarobotImageDeclaration
		for _, key := range keys {
			items = append(items, chartutil.DatarobotImageDeclaration{Name: key, Image: uniqueEntries[key]})
		}

		yamlItems, err := yaml.Marshal(items)
		if err != nil {
			return fmt.Errorf("Error converting to YAML: %v\n", err)
		}

		output := map[string]interface{}{
			"annotations": map[string]string{
				string(annotation): string(yamlItems),
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

type generateInput struct {
	Values     []string
	ValueFiles []string
	Debug      bool
}

var g generateInput

func init() {
	rootCmd.AddCommand(generateCmd)
	generateCmd.Flags().StringVarP(&annotation, "annotation", "a", "datarobot.com/images", "annotation to lookup")
	generateCmd.Flags().BoolVarP(&g.Debug, "debug", "d", false, "debug")
	generateCmd.Flags().StringSliceVarP(&g.ValueFiles, "values", "f", []string{}, "specify values in a YAML file or a URL (can specify multiple)")
	generateCmd.Flags().StringArrayVar(&g.Values, "set", []string{}, "set values on the command line (can specify multiple or separate values with commas: key1=val1,key2=val2)")
}
