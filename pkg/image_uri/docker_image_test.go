package image_uri

import (
	"testing"
)

func TestParseDockerImage(t *testing.T) {
	tests := []struct {
		image     string
		expected  DockerUri
		expectErr bool
	}{
		{
			image: "nginx:1.19",
			expected: DockerUri{
				RegistryHost: "docker.io",
				Organization: "",
				Project:      "",
				ImageName:    "nginx",
				Tag:          "1.19",
			},
			expectErr: false,
		},
		{
			image: "myregistry.com/myrepo/myimage:latest",
			expected: DockerUri{
				RegistryHost: "myregistry.com",
				Organization: "myrepo",
				Project:      "",
				ImageName:    "myimage",
				Tag:          "latest",
			},
			expectErr: false,
		},
		{
			image: "myregistry.com/myrepo/myproject/myimage:latest",
			expected: DockerUri{
				RegistryHost: "myregistry.com",
				Organization: "myrepo",
				Project:      "myproject",
				ImageName:    "myimage",
				Tag:          "latest",
			},
			expectErr: false,
		},
		{
			image: "myregistry.com/myrepo/myproject1/myproject2/myimage:latest",
			expected: DockerUri{
				RegistryHost: "myregistry.com",
				Organization: "myrepo",
				Project:      "myproject1/myproject2",
				ImageName:    "myimage",
				Tag:          "latest",
			},
			expectErr: false,
		},
		{
			image: "myimage:latest",
			expected: DockerUri{
				RegistryHost: "docker.io",
				Project:      "",
				ImageName:    "myimage",
				Tag:          "latest",
			},
			expectErr: false,
		},
		{
			image: "myimage",
			expected: DockerUri{
				RegistryHost: "docker.io",
				Project:      "",
				ImageName:    "myimage",
				Tag:          "",
			},
			expectErr: false,
		},
	}

	for _, test := range tests {
		result, err := NewDockerUri(test.image)

		if test.expectErr {
			if err == nil {
				t.Errorf("expected an error for image %s, got none", test.image)
			}
		} else {
			if err != nil {
				t.Errorf("did not expect an error for image %s, got: %v", test.image, err)
			}
			if result != test.expected {
				t.Errorf("for image %s, expected %+v, got %+v", test.image, test.expected, result)
			}
		}
	}
}
