package rackspace

import (
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/rackspace/gophercloud"
	"github.com/rackspace/gophercloud/rackspace/blockstorage/v1/volumes"
)

// VolumeStateRefreshFunc returns a resource.StateRefreshFunc that is used to watch
// an OpenStack volume.
func VolumeStateRefreshFunc(client *gophercloud.ServiceClient, volumeID string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		v, err := volumes.Get(client, volumeID).Extract()
		if err != nil {
			errCode, ok := err.(*gophercloud.UnexpectedResponseCodeError)
			if !ok {
				return nil, "", err
			}
			if errCode.Actual == 404 {
				return v, "deleted", nil
			}
			return nil, "", err
		}

		return v, v.Status, nil
	}
}
