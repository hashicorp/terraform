package openstack

import (
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/subnetpools"
	"github.com/hashicorp/terraform/helper/resource"
)

func networkingSubnetpoolV2StateRefreshFunc(client *gophercloud.ServiceClient, id string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		subnetpool, err := subnetpools.Get(client, id).Extract()
		if err != nil {
			if _, ok := err.(gophercloud.ErrDefault404); ok {
				return subnetpool, "DELETED", nil
			}
			if errCode, ok := err.(gophercloud.ErrUnexpectedResponseCode); ok {
				if errCode.Actual == 409 {
					return subnetpool, "ACTIVE", nil
				}
			}

			return nil, "", err
		}

		return subnetpool, "ACTIVE", nil
	}
}
