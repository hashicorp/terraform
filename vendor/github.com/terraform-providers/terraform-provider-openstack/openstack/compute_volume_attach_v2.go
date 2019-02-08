package openstack

import (
	"fmt"
	"log"
	"strings"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/volumeattach"

	"github.com/hashicorp/terraform/helper/resource"
)

func computeVolumeAttachV2ParseID(id string) (string, string, error) {
	parts := strings.Split(id, "/")
	if len(parts) < 2 {
		return "", "", fmt.Errorf("unable to determine openstack_compute_volume_attach_v2 ID")
	}

	instanceID := parts[0]
	attachmentID := parts[1]

	return instanceID, attachmentID, nil
}

func computeVolumeAttachV2AttachFunc(
	computeClient *gophercloud.ServiceClient, instanceId, attachmentId string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		va, err := volumeattach.Get(computeClient, instanceId, attachmentId).Extract()
		if err != nil {
			if _, ok := err.(gophercloud.ErrDefault404); ok {
				return va, "ATTACHING", nil
			}
			return va, "", err
		}

		return va, "ATTACHED", nil
	}
}

func computeVolumeAttachV2DetachFunc(
	computeClient *gophercloud.ServiceClient, instanceId, attachmentId string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		log.Printf("[DEBUG] openstack_compute_volume_attach_v2 attempting to detach OpenStack volume %s from instance %s",
			attachmentId, instanceId)

		va, err := volumeattach.Get(computeClient, instanceId, attachmentId).Extract()
		if err != nil {
			if _, ok := err.(gophercloud.ErrDefault404); ok {
				return va, "DETACHED", nil
			}
			return va, "", err
		}

		err = volumeattach.Delete(computeClient, instanceId, attachmentId).ExtractErr()
		if err != nil {
			if _, ok := err.(gophercloud.ErrDefault404); ok {
				return va, "DETACHED", nil
			}

			if _, ok := err.(gophercloud.ErrDefault400); ok {
				return nil, "", nil
			}

			return nil, "", err
		}

		log.Printf("[DEBUG] openstack_compute_volume_attach_v2 (%s/%s) is still active.", instanceId, attachmentId)
		return nil, "", nil
	}
}
