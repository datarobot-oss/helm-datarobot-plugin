package cmd

import (
	"fmt"
	"strings"

	"github.com/datarobot-oss/helm-datarobot-plugin/pkg/crdchart"
	"github.com/datarobot-oss/helm-datarobot-plugin/pkg/render_helper"
	"github.com/spf13/cobra"
)

type crdChartInput struct {
	Namespace   string
	ReleaseName string
	ValueFiles  []string
	Values      []string
	KubeVersion string
	APIVersions []string
	Output      string
	KeepCRDs    bool
	Debug       bool
}

var cc crdChartInput

var crdChartCmd = &cobra.Command{
	Use:          "crd-chart <prime-chart.tgz>",
	Short:        "extract CRDs from a chart into a standalone datarobot-infra chart",
	SilenceUsage: true,
	Long: strings.Replace(`

Render a chart (e.g. datarobot-prime) and extract only its
CustomResourceDefinitions into a standalone, installable datarobot-infra chart.

Example:
'''sh
$ helm datarobot crd-chart datarobot-prime.tgz -o datarobot-infra.tgz
'''`, "'", "`", -1),
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		chartPath := args[0]

		src, err := crdchart.ReadSourceChartMeta(chartPath)
		if err != nil {
			return err
		}
		if src.Version == "" {
			cmd.PrintErrln("warning: source chart has no version; stamping 0.0.0")
			src.Version = "0.0.0"
		}

		// Force CRD emission: IncludeCRDs renders crds/-dir CRDs (Envoy);
		// global.installCRDs re-enables the templated DR CRDs that a
		// production values.yaml may have disabled. Appended AFTER the
		// user's --set so these win.
		//
		// installCRDs and keepCRDs are deliberately asymmetric: installCRDs is
		// always forced true because it gates whether the CRD is EMITTED at all
		// (we need it emitted to extract it, regardless of --keep-crds). keepCRDs
		// mirrors the --keep-crds flag because it only gates the
		// helm.sh/resource-policy: keep ANNOTATION some source charts bake in at
		// render time. Don't "simplify" these back to both being forced true --
		// that reintroduces CRD-001 (the annotation leaking into --keep-crds=false
		// output).
		forced := append([]string{}, cc.Values...)
		forced = append(forced, "global.installCRDs=true",
			fmt.Sprintf("global.keepCRDs=%t", cc.KeepCRDs))

		manifest, err := render_helper.RenderChartWithOptions(chartPath, cc.ValueFiles, forced, &render_helper.RenderOptions{
			Namespace:   cc.Namespace,
			ReleaseName: cc.ReleaseName,
			KubeVersion: cc.KubeVersion,
			IncludeCRDs: true,
			APIVersions: cc.APIVersions,
		})
		if err != nil {
			return err
		}

		resources, err := crdchart.ParseManifests(manifest)
		if err != nil {
			return err
		}
		crds := crdchart.KeepMatching(resources, crdchart.IsCRD)

		crds, collisions := crdchart.DedupeByName(crds)
		for _, name := range collisions {
			cmd.PrintErrf("warning: duplicate CRD %q across subcharts; keeping last\n", name)
		}

		if len(crds) == 0 {
			return fmt.Errorf("no CRDs found — check values/flags")
		}

		for i, r := range crds {
			if cc.KeepCRDs {
				r, err = crdchart.AddKeepAnnotation(r)
			} else {
				r, err = crdchart.StripKeepAnnotation(r)
			}
			if err != nil {
				return fmt.Errorf("failed to annotate CRD %s: %w", r.Name, err)
			}
			r.RawYAML = crdchart.EscapeBraces(r.RawYAML)
			crds[i] = r
			if cc.Debug {
				cmd.Printf("CRD: %s\n", r.Name)
			}
		}

		outPath := cc.Output
		if outPath == "" {
			outPath = fmt.Sprintf("./datarobot-infra-%s.tgz", src.Version)
		}

		builtChart, err := crdchart.BuildChart(crds, src)
		if err != nil {
			return err
		}
		if err := crdchart.PackageChart(builtChart, outPath); err != nil {
			return err
		}

		cmd.Printf("Extracted %d CRDs -> %s\n", len(crds), outPath)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(crdChartCmd)
	crdChartCmd.Flags().StringVar(&cc.Namespace, "namespace", "datarobot", "render namespace (.Release.Namespace)")
	crdChartCmd.Flags().StringVar(&cc.ReleaseName, "release-name", "datarobot", ".Release.Name")
	crdChartCmd.Flags().StringSliceVarP(&cc.ValueFiles, "values", "f", []string{}, "specify values in a YAML file (can specify multiple)")
	crdChartCmd.Flags().StringArrayVar(&cc.Values, "set", []string{}, "set values on the command line (can specify multiple)")
	crdChartCmd.Flags().StringVar(&cc.KubeVersion, "kube-version", "v1.32.0", "Helm template KubeVersion")
	crdChartCmd.Flags().StringSliceVar(&cc.APIVersions, "api-versions", []string{}, "extra API versions for rendering (can specify multiple)")
	crdChartCmd.Flags().StringVarP(&cc.Output, "output", "o", "", "output .tgz path (default ./datarobot-infra-<srcVersion>.tgz)")
	crdChartCmd.Flags().BoolVar(&cc.KeepCRDs, "keep-crds", true, "add helm.sh/resource-policy: keep annotation")
	crdChartCmd.Flags().BoolVarP(&cc.Debug, "debug", "d", false, "verbose per-CRD listing")
}
