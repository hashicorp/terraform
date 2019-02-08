package openstack

import (
	"fmt"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/provider"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/networks"
	"github.com/gophercloud/gophercloud/pagination"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

// networkingNetworkV2ID retrieves network ID by the provided name.
func networkingNetworkV2ID(d *schema.ResourceData, meta interface{}, networkName string) (string, error) {
	config := meta.(*Config)
	networkingClient, err := config.networkingV2Client(GetRegion(d, config))
	if err != nil {
		return "", fmt.Errorf("Error creating OpenStack network client: %s", err)
	}

	opts := networks.ListOpts{Name: networkName}
	pager := networks.List(networkingClient, opts)
	networkID := ""

	err = pager.EachPage(func(page pagination.Page) (bool, error) {
		networkList, err := networks.ExtractNetworks(page)
		if err != nil {
			return false, err
		}

		for _, n := range networkList {
			if n.Name == networkName {
				networkID = n.ID
				return false, nil
			}
		}

		return true, nil
	})

	return networkID, err
}

// networkingNetworkV2Name retrieves network name by the provided ID.
func networkingNetworkV2Name(d *schema.ResourceData, meta interface{}, networkID string) (string, error) {
	config := meta.(*Config)
	networkingClient, err := config.networkingV2Client(GetRegion(d, config))
	if err != nil {
		return "", fmt.Errorf("Error creating OpenStack network client: %s", err)
	}

	opts := networks.ListOpts{ID: networkID}
	pager := networks.List(networkingClient, opts)
	networkName := ""

	err = pager.EachPage(func(page pagination.Page) (bool, error) {
		networkList, err := networks.ExtractNetworks(page)
		if err != nil {
			return false, err
		}

		for _, n := range networkList {
			if n.ID == networkID {
				networkName = n.Name
				return false, nil
			}
		}

		return true, nil
	})

	return networkName, err
}

func resourceNetworkingNetworkV2StateRefreshFunc(client *gophercloud.ServiceClient, networkID string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		n, err := networks.Get(client, networkID).Extract()
		if err != nil {
			if _, ok := err.(gophercloud.ErrDefault404); ok {
				return n, "DELETED", nil
			}
			if errCode, ok := err.(gophercloud.ErrUnexpectedResponseCode); ok {
				if errCode.Actual == 409 {
					return n, "ACTIVE", nil
				}
			}

			return n, "", err
		}

		return n, n.Status, nil
	}
}

func expandNetworkingNetworkSegmentsV2(segments *schema.Set) []provider.Segment {
	rawSegments := segments.List()

	providerSegments := make([]provider.Segment, len(rawSegments))
	for i, raw := range rawSegments {
		rawMap := raw.(map[string]interface{})
		providerSegments[i] = provider.Segment{
			PhysicalNetwork: rawMap["physical_network"].(string),
			NetworkType:     rawMap["network_type"].(string),
			SegmentationID:  rawMap["segmentation_id"].(int),
		}
	}

	return providerSegments
}
