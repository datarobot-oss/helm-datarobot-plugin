package cmd

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/datarobot-oss/helm-datarobot-plugin/pkg/render_helper"
	"github.com/spf13/cobra"
)

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
		manifest, err := render_helper.NewRenderItems(chartPath)
		if err != nil {
			return fmt.Errorf("Error loading chart %s: %v", chartPath, err)
		}

		imageDoc, err := ExtractImagesFromCharts(args)
		if err != nil {
			return fmt.Errorf("Error ExtractImagesFromCharts: %v", err)
		}
		if validateDebug {
			fmt.Printf("---\n# annotation: %s\n", annotation)
			fmt.Printf("---\n# imageDoc: %s\n", imageDoc)
		}

		if len(imageDoc) == 0 {
			return fmt.Errorf("imageDoc is empty")
		}

		for fileName, template := range manifest {
			// We only apply the following lint rules to yaml files
			if filepath.Ext(fileName) != ".yaml" || filepath.Ext(fileName) == ".yml" {
				continue
			}

			if validateDebug {
				fmt.Printf("---\n# Source: %s\n%s\n", fileName, template)
			}

			manifestImages, err := ExtractImagesFromManifest(template)
			if err != nil {
				return fmt.Errorf("Error ExtractImagesFromManifest chart %s: %v", fileName, err)
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
