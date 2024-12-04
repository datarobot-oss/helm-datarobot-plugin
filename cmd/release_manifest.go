package cmd

import (
	"errors"
	"fmt"
	"strings"

	"gopkg.in/yaml.v2"
	"helm.sh/helm/v3/pkg/chart/loader"

	"github.com/datarobot-oss/helm-datarobot-plugin/pkg/chartutil"
	"github.com/datarobot-oss/helm-datarobot-plugin/pkg/image_uri"
	"github.com/spf13/cobra"
)

const ARCHIVE_EXT = ".tar.zst"
const REL_ORG = "datarobot"

type releaseManifestOutput struct {
	Images map[string]releaseManifestImage `yaml:"images"`
}

type releaseManifestImage struct {
	Source string `yaml:"source"`
	Name   string `yaml:"name"`
	Tag    string `yaml:"tag"`
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
			Name:   iUri.SetOrg(REL_ORG),
			Tag:    imageTag,
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

func ExtractImagesFromCharts(args []string) ([]chartutil.DatarobotImageDeclaration, error) {
	allChartImages := make([]chartutil.ChartImages, 0)
	for _, chartPath := range args {
		c, err := loader.Load(chartPath)
		if err != nil {
			return nil, fmt.Errorf("Error loading chart %s: %v", chartPath, err)
		}
		allChartImages = append(
			allChartImages,
			chartutil.RecursiveRenderDatarobotImages(c, annotation)...,
		)
	}
	allImages := make([]chartutil.DatarobotImageDeclaration, 0)
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

func generateReleaseManifest(args []string) (map[string]releaseManifestImage, error) {
	allImages, err := ExtractImagesFromCharts(args)
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
$ helm datarobot release-manifest testdata/test-chart1/
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

var skipDuplicated bool

func init() {
	rootCmd.AddCommand(releaseManifestCmd)
	releaseManifestCmd.Flags().StringVarP(&annotation, "annotation", "a", "datarobot.com/images", "annotation to lookup")
	releaseManifestCmd.Flags().BoolVarP(&skipDuplicated, "skip-duplicated", "", false, "skip duplicated images")
}
