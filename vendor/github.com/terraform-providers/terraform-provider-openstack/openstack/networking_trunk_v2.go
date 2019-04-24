package openstack

import (
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/trunks"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func networkingTrunkV2StateRefreshFunc(client *gophercloud.ServiceClient, trunkID string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		trunk, err := trunks.Get(client, trunkID).Extract()
		if err != nil {
			if _, ok := err.(gophercloud.ErrDefault404); ok {
				return trunk, "DELETED", nil
			}

			return nil, "", err
		}

		return trunk, trunk.Status, nil
	}
}

func flattenNetworkingTrunkV2Subports(subports []trunks.Subport) []map[string]interface{} {
	trunkSubports := make([]map[string]interface{}, len(subports))

	for i, subport := range subports {
		trunkSubports[i] = map[string]interface{}{
			"port_id":           subport.PortID,
			"segmentation_type": subport.SegmentationType,
			"segmentation_id":   subport.SegmentationID,
		}
	}

	return trunkSubports
}

func expandNetworkingTrunkV2Subports(subports *schema.Set) []trunks.Subport {
	rawSubports := subports.List()

	trunkSubports := make([]trunks.Subport, len(rawSubports))
	for i, raw := range rawSubports {
		rawMap := raw.(map[string]interface{})

		trunkSubports[i] = trunks.Subport{
			PortID:           rawMap["port_id"].(string),
			SegmentationType: rawMap["segmentation_type"].(string),
			SegmentationID:   rawMap["segmentation_id"].(int),
		}
	}

	return trunkSubports
}

func expandNetworkingTrunkV2SubportsRemove(subports *schema.Set) []trunks.RemoveSubport {
	rawSubports := subports.List()

	subportsToRemove := make([]trunks.RemoveSubport, len(rawSubports))
	for i, raw := range rawSubports {
		rawMap := raw.(map[string]interface{})

		subportsToRemove[i] = trunks.RemoveSubport{
			PortID: rawMap["port_id"].(string),
		}
	}

	return subportsToRemove
}
