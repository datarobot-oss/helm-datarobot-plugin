package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

var imageCmd = &cobra.Command{
	Use:     "images",
	Aliases: []string{"im", "image", "img", "imgs"},
	Short:   "list images from a given chart",
	Long: strings.Replace(`
DataRobot introduced a custom annotation 'datarobot.com/images' to solve
this problem. This annotation lets chart developers manifest which images are
required by the application. Those images will be included into enterprise
releases automatically.

Example:
'''sh
$ yq ".annotations" tests/charts/test-chart1/Chart.yaml
datarobot.com/images: |
- name: test-image1
image: docker.io/datarobotdev/test-image1:{{.Chart.AppVersion}}
'''

The value of 'datarobot.com/images' annotation is a template (pay attention to
'|') that is going to be rendered with 'gotpl' (just like everything else in
Helm) with '.Chart' available in the context.

Subcommand 'images' parses, combines and returns 'datarobot.com/images'
annotations of a chart and its subcharts, e.g.:

'''sh
$ helm datarobot images tests/charts/test-chart1
- name: test-image1
image: docker.io/datarobotdev/test-image1:1.0.0
- name: test-image2
image: docker.io/datarobotdev/test-image2:2.0.0
- name: test-image3
image: docker.io/datarobotdev/test-image3:3.0.0
'''

'''`, "'", "`", -1),
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		allImages, err := ExtractImagesFromCharts(args)
		if err != nil {
			return fmt.Errorf("Error ExtractImagesFromCharts: %v", err)
		}
		yamlData, err := yaml.Marshal(&allImages)
		if err != nil {
			return fmt.Errorf("Error writing yaml: %v", err)
		}

		stdout := cmd.OutOrStdout()
		stdout.Write(yamlData)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(imageCmd)
	imageCmd.Flags().StringVarP(&annotation, "annotation", "a", "datarobot.com/images", "annotation to lookup")
}
