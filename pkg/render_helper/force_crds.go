package render_helper

import (
	"fmt"
	"sort"
	"strings"

	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/cli/values"
	"helm.sh/helm/v3/pkg/getter"
)

// ForceInstallCRDs walks a chart's dependency tree and returns the extra --set
// entries needed to force every CRD emission gate on, regardless of what the
// user's values resolve to.
//
// Prime's CRDs are gated by `and(.Values.installCRDs, .Values.global.installCRDs)`
// in each subchart's templates/. Rendering with IncludeCRDs only captures the
// crds/ directory (ungated), NOT these templated CRDs, so any values file or
// --set that flips either half to false silently ships an incomplete infra
// chart. Forcing both halves closes that hole.
//
// The returned `forced` slice MUST be appended AFTER the user's --set so it
// wins (Helm applies --set after value files, later entries overriding earlier
// ones). `overrides` names each installCRDs key the user explicitly set to
// false, so the caller can log the override instead of dropping CRDs silently.
func ForceInstallCRDs(chartPath string, valueFiles, setValues []string) (forced []string, overrides []string, err error) {
	loadedChart, err := loader.Load(chartPath)
	if err != nil {
		return nil, nil, fmt.Errorf("Error loading chart %s: %v", chartPath, err)
	}

	// Merge the user's values so we can detect which installCRDs keys they set
	// to false and report them.
	settings := cli.New()
	merged, err := (&values.Options{ValueFiles: valueFiles, Values: setValues}).MergeValues(getter.All(settings))
	if err != nil {
		return nil, nil, err
	}

	// Global half — shared by every subchart.
	forced = append(forced, "global.installCRDs=true")
	if valueIsFalse(merged, "global.installCRDs") {
		overrides = append(overrides, "global.installCRDs")
	}

	// Local half — every subchart's own installCRDs, recursively. enabled=false
	// subcharts are still walked (loader loads all charts/ members); forcing
	// their installCRDs is harmless because the enabled condition still gates
	// rendering, so a disabled subchart's CRD stays excluded.
	var walk func(deps []*chart.Chart, prefix string)
	walk = func(deps []*chart.Chart, prefix string) {
		for _, d := range deps {
			path := d.Name()
			if prefix != "" {
				path = prefix + "." + d.Name()
			}
			forced = append(forced, path+".installCRDs=true")
			if valueIsFalse(merged, path+".installCRDs") {
				overrides = append(overrides, path+".installCRDs")
			}
			walk(d.Dependencies(), path)
		}
	}
	walk(loadedChart.Dependencies(), "")

	sort.Strings(overrides) // deterministic log order
	return forced, overrides, nil
}

// valueIsFalse reports whether a dotted path in the merged values resolves to a
// literal boolean false. A missing path (or a non-bool) is not "false".
func valueIsFalse(m map[string]interface{}, dottedPath string) bool {
	var cur interface{} = m
	for _, seg := range strings.Split(dottedPath, ".") {
		asMap, ok := cur.(map[string]interface{})
		if !ok {
			return false
		}
		cur, ok = asMap[seg]
		if !ok {
			return false
		}
	}
	b, ok := cur.(bool)
	return ok && !b
}
