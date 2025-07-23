package cmd

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/datarobot-oss/helm-datarobot-plugin/pkg/chartutil"
	"github.com/datarobot-oss/helm-datarobot-plugin/pkg/image_uri"
	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

const ARCHIVE_EXT = ".tar.zst"

type releaseManifestOutput struct {
	Images map[string]releaseManifestImage `yaml:"images"`
}

type releaseManifestImage struct {
	Source string            `yaml:"source"`
	Name   string            `yaml:"name"`
	Tag    string            `yaml:"tag"`
	Labels map[string]string `yaml:"labels,omitempty"`
}

func getReleaseManifest(images []chartutil.DatarobotImageDeclaration, skipDuplicated bool) (map[string]releaseManifestImage, error) {
	result := make(map[string]releaseManifestImage)
	for _, image := range images {
		iUri, err := image_uri.NewDockerUri(image.Image)
		if err != nil {
			return nil, err
		}

		imageTag := image.Tag
		if imageTag == "" {
			imageTag = iUri.Tag
		}
		rmi := releaseManifestImage{
			Source: iUri.String(),
			Name:   iUri.Base(),
			Tag:    imageTag,
		}

		if len(addLabels) > 0 || addAllLabels {
			allLabels, err := ExtractLabels(iUri.String())
			if err != nil {
				log.Fatalf("Error extracting labels: %v", err)
			}
			reqLabel := make(map[string]string)
			if len(allLabels) > 0 {
				for _, label := range addLabels {
					if value, exists := allLabels[label]; exists {
						reqLabel[label] = value
					}
				}
			}

			if addAllLabels {
				rmi.Labels = allLabels
			} else {
				rmi.Labels = reqLabel
			}

		}

		archiveName := image.Name + ARCHIVE_EXT
		_, archiveNameExists := result[archiveName]
		if archiveNameExists {
			if skipDuplicated {
				fmt.Printf("[Warning] Duplicate image name: %s\n", image.Name)
			} else {
				err := fmt.Errorf("Duplicate image name: %s", image.Name)
				return nil, err
			}

		}

		result[archiveName] = rmi
	}
	return result, nil
}

// It fetches the image configuration metadata without pulling the full image.
func ExtractLabels(imageName string) (map[string]string, error) {
	// Get the raw configuration JSON from the registry
	configJSON, err := crane.Config(imageName)
	if err != nil {
		return nil, fmt.Errorf("failed to get image config: %w", err)
	}

	// Parse the JSON into a map
	var configData map[string]interface{}
	if err := json.Unmarshal(configJSON, &configData); err != nil {
		return nil, fmt.Errorf("failed to parse image config JSON: %w", err)
	}

	// Navigate to the labels in the config
	config, ok := configData["config"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("failed to extract 'config' section from image config")
	}

	labels, ok := config["Labels"].(map[string]interface{})
	if !ok || labels == nil {
		// Return an empty map if no labels are found
		return map[string]string{}, nil
	}

	// Convert labels to a string map
	labelMap := make(map[string]string)
	for key, value := range labels {
		if strValue, ok := value.(string); ok {
			labelMap[key] = strValue
		}
	}

	return labelMap, nil
}

func deduplicate(images []chartutil.DatarobotImageDeclaration) []chartutil.DatarobotImageDeclaration {
	seen := make(map[string]bool)
	result := make([]chartutil.DatarobotImageDeclaration, 0)

	for _, image := range images {
		if !seen[image.Image] {
			result = append(result, image)
			seen[image.Image] = true
		}
	}

	return result
}

func generateReleaseManifest(args []string) (map[string]releaseManifestImage, error) {
	allImages, err := chartutil.ExtractImagesFromCharts(args, annotation)
	if err != nil {
		return nil, fmt.Errorf("Error ExtractImagesFromCharts: %v", err)
	}
	releaseManifest, err := getReleaseManifest(deduplicate(allImages), skipDuplicated)
	if err != nil {
		return nil, fmt.Errorf("Error generating release manifest: %v", err)
	}
	return releaseManifest, nil
}

var releaseManifestCmd = &cobra.Command{
	Use:     "release-manifest",
	Aliases: []string{"rel"},
	Short:   "release-manifest",
	Long: strings.Replace(`
Subcommand 'release-manifest' is conceptually similar to subcommand 'images'.
it supports more than 1 chart, so we can produce a single manifest and other umbrella charts.

Example:
'''sh
$ helm datarobot release-manifest tests/charts/test-chart1/
images:
	test-image1.tar.zst:
		source: docker.io/datarobotdev/test-image1:1.0.0
		name: docker.io/datarobot/test-image1
		tag: 1.0.0
	test-image2.tar.zst:
		source: docker.io/datarobotdev/test-image2:2.0.0
		name: docker.io/datarobot/test-image2
		tag: 2.0.0
	test-image3.tar.zst:
		source: docker.io/datarobotdev/test-image3:3.0.0
		name: docker.io/datarobot/test-image3
		tag: 3.0.0
'''

'''`, "'", "`", -1),
	Args: cobra.MinimumNArgs(1), // Requires at least one argument (file path)
	RunE: func(cmd *cobra.Command, args []string) error {

		releaseManifest, err := generateReleaseManifest(args)
		if err != nil {
			return fmt.Errorf("Error generateReleaseManifest: %v", err)
		}
		output := releaseManifestOutput{Images: releaseManifest}
		yamlData, err := yaml.Marshal(&output)
		if err != nil {
			return fmt.Errorf("Error writing yaml: %v", err)
		}

		stdout := cmd.OutOrStdout()
		stdout.Write(yamlData)
		return nil
	},
}

var skipDuplicated, addAllLabels bool
var addLabels []string

func init() {
	rootCmd.AddCommand(releaseManifestCmd)
	releaseManifestCmd.Flags().StringVarP(&annotation, "annotation", "a", "datarobot.com/images", "annotation to lookup")
	releaseManifestCmd.Flags().BoolVarP(&skipDuplicated, "skip-duplicated", "", false, "skip duplicated images")
	releaseManifestCmd.Flags().BoolVarP(&addAllLabels, "all-labels", "", false, "add all labes")
	releaseManifestCmd.Flags().StringArrayVarP(&addLabels, "label", "l", []string{}, "Specify labels (can be used multiple times)")
}
