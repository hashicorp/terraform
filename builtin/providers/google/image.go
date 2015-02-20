package google

import (
	"fmt"
	"strings"
)

// If the given name is a URL, return it.
// If it is of the form project/name, use that URL.
// If it is of the form name then look in the configured project and then hosted image projects.
func resolveImage(c *Config, name string) (string, error) {

	if strings.HasPrefix(name, "https://www.googleapis.com/compute/v1/") {
		return name, nil

	} else {
		splitName := strings.Split(name, "/")
		if len(splitName) == 1 {

			// Must infer the project name:

			// First, try the configured project.
			image, err := c.clientCompute.Images.Get(c.Project, name).Do()
			if err == nil {
				return image.SelfLink, nil
			}

			// If we match a lookup for an alternate project, then try that next.
			// If not, we return the original error.

			// If the image name contains the left hand side, we use the project from the right hand
			// side.
			imageMap := map[string]string{
				"centos":   "centos-cloud",
				"coreos":   "coreos-cloud",
				"debian":   "debian-cloud",
				"opensuse": "opensuse-cloud",
				"rhel":     "rhel-cloud",
				"sles":     "suse-cloud",
				"ubuntu":   "ubuntu-os-cloud",
				"windows":  "windows-cloud",
			}
			var project string
			for k, v := range imageMap {
				if strings.Contains(name, k) {
					project = v
					break
				}
			}
			if project == "" {
				return "", err
			}

			// There was a match, but the image still may not exist, so check it:
			image, err = c.clientCompute.Images.Get(project, name).Do()
			if err == nil {
				return image.SelfLink, nil
			}

			return "", err

		} else if len(splitName) == 2 {
			image, err := c.clientCompute.Images.Get(splitName[0], splitName[1]).Do()
			if err == nil {
				return image.SelfLink, nil
			}
			return "", err

		} else {
			return "", fmt.Errorf("Invalid image name, require URL, project/name, or just name: %s", name)
		}
	}

}
