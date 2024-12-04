package image_uri

import (
	"fmt"
	"regexp"
	"strings"
)

// DockerImage represents a parsed Docker image URL.
type DockerUri struct {
	RegistryHost string
	Organization string
	Project      string
	ImageName    string
	Tag          string
}

// ParseDockerImage parses a Docker image URL and extracts its components.
func NewDockerUri(imageURL string) (DockerUri, error) {
	// Define a regex pattern to match the Docker image URL
	pattern := `^(?:(?P<registry>[^/]+)/)?(?:(?P<org>[^/]+)/)?(?:(?P<projects>.+?)/)?(?P<imageName>[^:]+)(?::(?P<tag>.+))?$`
	re := regexp.MustCompile(pattern)
	dockerImage := DockerUri{
		RegistryHost: "docker.io", // Default registry
		Tag:          "latest",    // Default tag
	}

	// Match the image URL against the regex pattern
	matches := re.FindStringSubmatch(imageURL)
	if matches == nil {
		return dockerImage, fmt.Errorf("invalid Docker image URL: %s", imageURL)
	}

	groupNames := re.SubexpNames()
	result := make(map[string]string)
	for i, match := range matches {
		if i != 0 { // Skip the full match
			result[groupNames[i]] = match
		}
	}

	// Split the projects by '/' if it exists
	var projects []string
	if projectsStr, ok := result["projects"]; ok && projectsStr != "" {
		projects = regexp.MustCompile(`/`).Split(projectsStr, -1)
	}

	dockerImage.RegistryHost = result["registry"] // Optional registry host
	dockerImage.Organization = result["org"]      // Optional organization
	dockerImage.Project = strings.Join(projects, "/")
	dockerImage.ImageName = result["imageName"]
	dockerImage.Tag = result["tag"] // Optional tag

	// If the registry host is not provided, set it to "docker.io" (default)
	if dockerImage.RegistryHost == "" {
		dockerImage.RegistryHost = "docker.io"
	}

	if dockerImage.ImageName == "" && dockerImage.Project != "" {
		dockerImage.ImageName = dockerImage.Project
		dockerImage.Project = ""
	}

	if strings.Contains(dockerImage.ImageName, "/") {
		parts := strings.Split(dockerImage.ImageName, "/")
		dockerImage.ImageName = parts[len(parts)-1]
		dockerImage.Project += "/" + strings.Join(parts[:len(parts)-1], "/")
	}

	return dockerImage, nil
}

func (d *DockerUri) Base() string {
	return d.Join([]string{d.RegistryHost, d.Organization, d.Project, d.ImageName}, "/")
}

func (d *DockerUri) String() string {
	tag := ""
	if d.Tag != "" {
		tag = ":" + d.Tag
	}
	return fmt.Sprintf("%s%s", d.Base(), tag)
}

func (d *DockerUri) Join(s []string, delimit string) string {
	var r []string
	for _, str := range s {
		if str != "" {
			r = append(r, str)
		}
	}
	return strings.Join(r, delimit)
}

func (d *DockerUri) SetOrg(org string) string {
	d.Organization = org
	return d.Base()
}
