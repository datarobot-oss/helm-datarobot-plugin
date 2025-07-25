package cmd

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/datarobot-oss/helm-datarobot-plugin/pkg/chartutil"
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
		manifest, err := render_helper.RenderChart(chartPath, v.ValueFiles, v.Values)
		if err != nil {
			return fmt.Errorf("Error loading chart %s: %v", chartPath, err)
		}

		imageDoc, err := chartutil.ExtractImagesFromCharts(args, annotation)
		if err != nil {
			return fmt.Errorf("Error ExtractImagesFromCharts: %v", err)
		}
		if v.Debug {
			fmt.Printf("---\n# annotation: %s\n", annotation)
			b, err := json.MarshalIndent(imageDoc, "", "  ")
			if err == nil {
				fmt.Println(string(b))
			}
		}

		if len(imageDoc) == 0 {
			return fmt.Errorf("imageDoc is empty")
		}
		var errorImageAllowed []string
		for _, template := range strings.Split(manifest, "\n---\n") {
			if v.Debug {
				fmt.Printf("---\n%s\n", template)
			}

			manifestImages, err := ExtractImagesFromManifest(template)
			if err != nil {
				return fmt.Errorf("Error ExtractImagesFromManifest chart: %v", err)
			}
			// Validate manifestImages against the imageDoc
			for _, image := range manifestImages {
				if !isImageDeclared(image, imageDoc) {
					if !SliceHas(errorImageAllowed, image) {
						errorImageAllowed = append(errorImageAllowed, image)
					}
				}
			}

		}

		if len(errorImageAllowed) > 0 {
			sort.Strings(errorImageAllowed)
			return fmt.Errorf("Images not declared as ImageDoc:\n%s", strings.Join(errorImageAllowed, "\n"))
		} else {
			cmd.Print("Image Doc Valid")
		}

		return nil
	},
}

type validateInput struct {
	Values     []string
	ValueFiles []string
	Debug      bool
}

var v validateInput

func init() {
	rootCmd.AddCommand(validateCmd)
	validateCmd.Flags().StringVarP(&annotation, "annotation", "a", "datarobot.com/images", "annotation to lookup")
	validateCmd.Flags().BoolVarP(&v.Debug, "debug", "d", false, "debug")
	validateCmd.Flags().StringSliceVarP(&v.ValueFiles, "values", "f", []string{}, "specify values in a YAML file or a URL (can specify multiple)")
	validateCmd.Flags().StringArrayVar(&v.Values, "set", []string{}, "set values on the command line (can specify multiple or separate values with commas: key1=val1,key2=val2)")

}
