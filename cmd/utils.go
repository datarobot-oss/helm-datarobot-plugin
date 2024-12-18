package cmd

import (
	"fmt"
	"sort"
	"strings"

	dr_chartutil "github.com/datarobot-oss/helm-datarobot-plugin/pkg/chartutil"
	v1 "k8s.io/api/apps/v1"
	batch_v1 "k8s.io/api/batch/v1"
	"sigs.k8s.io/yaml"
)

// isImageDeclared checks if the image is in the declared imagedoc list
func isImageDeclared(image string, imageDoc []dr_chartutil.DatarobotImageDeclaration) bool {
	for _, im := range imageDoc {
		if strings.TrimSpace(image) == strings.TrimSpace(im.Image) {
			return true
		}
	}
	return false
}

func SliceHas(slice []string, str string) bool {
	for _, item := range slice {
		if item == str {
			return true
		}
	}
	return false
}

func ExtractImagesFromManifest(manifest string) ([]string, error) {
	var manifestImages []string

	var deployment v1.Deployment
	if err := yaml.Unmarshal([]byte(manifest), &deployment); err != nil {
		return nil, fmt.Errorf("Error unmarshalling YAML: %v\n", err)
	}
	// Collect images from init containers
	for _, initContainer := range deployment.Spec.Template.Spec.InitContainers {
		manifestImages = append(manifestImages, initContainer.Image)
	}

	// Collect images from regular containers
	for _, container := range deployment.Spec.Template.Spec.Containers {
		manifestImages = append(manifestImages, container.Image)
	}

	var statefulSet v1.StatefulSet
	if err := yaml.Unmarshal([]byte(manifest), &statefulSet); err != nil {
		return nil, fmt.Errorf("Error unmarshalling YAML: %v\n", err)
	}
	// Collect images from init containers
	for _, initContainer := range statefulSet.Spec.Template.Spec.InitContainers {
		manifestImages = append(manifestImages, initContainer.Image)
	}

	// Collect images from regular containers
	for _, container := range statefulSet.Spec.Template.Spec.Containers {
		manifestImages = append(manifestImages, container.Image)
	}

	var job batch_v1.Job
	if err := yaml.Unmarshal([]byte(manifest), &job); err != nil {
		return nil, fmt.Errorf("Error unmarshalling YAML: %v\n", err)
	}
	// Collect images from regular containers
	for _, container := range job.Spec.Template.Spec.Containers {
		manifestImages = append(manifestImages, container.Image)
	}

	var cronJob batch_v1.CronJob
	if err := yaml.Unmarshal([]byte(manifest), &cronJob); err != nil {
		return nil, fmt.Errorf("Error unmarshalling YAML: %v\n", err)
	}
	// Collect images from regular containers
	for _, container := range cronJob.Spec.JobTemplate.Spec.Template.Spec.Containers {
		manifestImages = append(manifestImages, container.Image)
	}

	sort.Strings(manifestImages)
	return manifestImages, nil
}
