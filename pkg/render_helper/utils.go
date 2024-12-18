package render_helper

import (
	"os"
	"path"
)

func isChartDirectory(dir string) bool {
	chartYamlPath := path.Join(dir, "Chart.yaml")
	_, err := os.Stat(chartYamlPath)
	return err == nil
}
