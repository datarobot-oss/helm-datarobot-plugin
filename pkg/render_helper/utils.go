package render_helper

import (
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/imdario/mergo" // For merging map
	"gopkg.in/yaml.v2"
)

func isChartDirectory(dir string) bool {
	chartYamlPath := path.Join(dir, "Chart.yaml")
	_, err := os.Stat(chartYamlPath)
	return err == nil
}

func loadValues(valuesFiles []string, setValues []string) (map[string]interface{}, error) {
	values := map[string]interface{}{}

	for _, file := range valuesFiles {
		fileValues := map[string]interface{}{}
		data, err := os.ReadFile(file)
		if err != nil {
			return nil, fmt.Errorf("failed to read values file %s: %v", file, err)
		}
		if err := yaml.Unmarshal(data, &fileValues); err != nil {
			return nil, fmt.Errorf("failed to unmarshal values file %s: %v", file, err)
		}
		if err := mergo.Merge(&values, fileValues, mergo.WithOverride); err != nil {
			return nil, fmt.Errorf("failed to merge values from file %s: %v", file, err)
		}
	}

	for _, setValue := range setValues {
		kv := strings.SplitN(setValue, "=", 2)
		if len(kv) != 2 {
			return nil, fmt.Errorf("invalid set value: %s", setValue)
		}
		keys := strings.Split(kv[0], ".")
		m := values
		for i, key := range keys {
			if i == len(keys)-1 {
				m[key] = kv[1]
			} else {
				if _, ok := m[key]; !ok {
					m[key] = map[string]interface{}{}
				}
				m = m[key].(map[string]interface{})
			}
		}
	}

	return values, nil
}
