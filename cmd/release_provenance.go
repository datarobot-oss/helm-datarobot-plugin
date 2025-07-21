package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/datarobot-oss/helm-datarobot-plugin/pkg/chartutil"
	"github.com/spf13/cobra"
)

// ProvenanceInfo holds image and its labels
// (If labels are nil, output as empty object)
type ProvenanceInfo struct {
	Image  string `json:"image"`
	Repo   string `json:"repo"`
	Commit string `json:"commit"`
}

var releaseProvenanceCmd = &cobra.Command{
	Use:   "release-provenance",
	Short: "Show image provenance (repo and commit) for all images in the chart",
	Long: strings.Replace(`
The "release-provenance" subcommand inspects all images declared in the chart's "datarobot.com/images"
annotation (and its subcharts), and attempts to extract provenance information for each image. Provenance
typically includes the source repository and the commit SHA or tag from which the image was built.

Example:

'''sh
$ helm-datarobot release-provenance datarobot-prime-11.0.0.tgz
[
  {
    "image": "docker.io/datarobotdev/test-service1:1.2.3",
    "repo": "test-repo1",
    "commit": "123abc456def7890123456789abcdef012345678"
  },
  {
    "image": "docker.io/datarobotdev/test-service2:4.5.6",
    "repo": "test-repo2",
    "commit": "abcdef1234567890abcdef1234567890abcdef12"
  },
  {
    "image": "docker.io/datarobotdev/test-service3:7.8.9",
    "repo": "test-repo3",
    "commit": "fedcba9876543210fedcba9876543210fedcba98"
  }
]
'''
`, "'", "`", -1),
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		images, err := chartutil.ExtractImagesFromCharts(args, annotation)
		if err != nil {
			return fmt.Errorf("error ExtractImagesFromCharts: %v", err)
		}
		var results []ProvenanceInfo
		for _, img := range images {
			labels, err := ExtractLabels(img.Image)
			if err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "Warning: failed to inspect %s: %v\n", img.Image, err)
				labels = map[string]string{}
			}
			results = append(results, ProvenanceInfo{
				Image:  img.Image,
				Repo:   labels["com.datarobot.repo-name"],
				Commit: labels["com.datarobot.repo-sha"],
			})
		}
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(results)
	},
}

func init() {
	rootCmd.AddCommand(releaseProvenanceCmd)
}
