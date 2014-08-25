package google

import (
	"strings"

	"code.google.com/p/google-api-go-client/compute/v1"
)

// readImage finds the image with the given name.
func readImage(c *Config, name string) (*compute.Image, error) {
	// First, always try ourselves first.
	image, err := c.clientCompute.Images.Get(c.Project, name).Do()
	if err == nil && image != nil && image.SelfLink != "" {
		return image, nil
	}

	// This is a map of names to the project name where a public image is
	// hosted. GCE doesn't have an API to simply look up an image without
	// a project so we do this jank thing.
	imageMap := map[string]string{
		"centos":   "centos-cloud",
		"coreos":   "coreos-cloud",
		"debian":   "debian-cloud",
		"opensuse": "opensuse-cloud",
		"rhel":     "rhel-cloud",
		"sles":     "suse-cloud",
	}

	// If we match a lookup for an alternate project, then try that next.
	// If not, we return the error.
	var project string
	for k, v := range imageMap {
		if strings.Contains(name, k) {
			project = v
			break
		}
	}
	if project == "" {
		return nil, err
	}

	return c.clientCompute.Images.Get(project, name).Do()
}
