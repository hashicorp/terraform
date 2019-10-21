package openstack

import (
	"github.com/gophercloud/gophercloud/openstack/compute/v2/flavors"
)

func expandComputeFlavorV2ExtraSpecs(raw map[string]interface{}) flavors.ExtraSpecsOpts {
	extraSpecs := make(flavors.ExtraSpecsOpts, len(raw))
	for k, v := range raw {
		extraSpecs[k] = v.(string)
	}

	return extraSpecs
}
