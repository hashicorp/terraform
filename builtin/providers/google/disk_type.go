package google

import (
	"google.golang.org/api/compute/v1"
)

// readDiskType finds the disk type with the given name.
func readDiskType(c *Config, zone *compute.Zone, name string) (*compute.DiskType, error) {
	diskType, err := c.clientCompute.DiskTypes.Get(c.Project, zone.Name, name).Do()
	if err == nil && diskType != nil && diskType.SelfLink != "" {
		return diskType, nil
	} else {
		return nil, err
	}
}
