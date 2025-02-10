package cmd

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net/http"
	"sort"
	"strings"
	"time"

	dr_chartutil "github.com/datarobot-oss/helm-datarobot-plugin/pkg/chartutil"
	v1 "k8s.io/api/apps/v1"
	batch_v1 "k8s.io/api/batch/v1"
	"sigs.k8s.io/yaml"
)

func GetTransport(caCertPath, certPath, keyPath string, insecureSkipVerify bool) (*http.Transport, error) {
	// Load custom CA certificate
	var caCertPool *x509.CertPool
	if caCertPath != "" {
		caCert, err := ioutil.ReadFile(caCertPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA certificate: %v", err)
		}

		caCertPool = x509.NewCertPool()
		if ok := caCertPool.AppendCertsFromPEM(caCert); !ok {
			return nil, fmt.Errorf("failed to append CA certificate")
		}
	}

	// Load client certificate and key
	var clientCert tls.Certificate
	if certPath != "" && keyPath != "" {
		var err error
		clientCert, err = tls.LoadX509KeyPair(certPath, keyPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load client certificate and key: %v", err)
		}
	}

	// Create and return a custom HTTP transport
	return &http.Transport{
		TLSClientConfig: &tls.Config{
			RootCAs:            caCertPool,
			Certificates:       []tls.Certificate{clientCert},
			InsecureSkipVerify: insecureSkipVerify,
		},
	}, nil
}

func checkRegistryOnline(url, username, password string) error {
	client := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	for i := 0; i < 5; i++ {
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return err
		}

		req.SetBasicAuth(username, password)

		resp, err := client.Do(req)
		if err != nil {
			fmt.Printf("Attempt %d: Failed to reach registry: %v\n", i+1, err)
			time.Sleep(2 * time.Second)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			fmt.Println("Registry is online.")
			return nil
		}

		fmt.Printf("Attempt %d: Registry returned status code %d\n", i+1, resp.StatusCode)
		time.Sleep(2 * time.Second)
	}

	return fmt.Errorf("registry is not online after 5 attempts")
}

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
