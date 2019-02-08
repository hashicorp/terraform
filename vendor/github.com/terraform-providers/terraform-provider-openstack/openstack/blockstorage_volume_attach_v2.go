package openstack

import (
	"fmt"
	"strings"

	"github.com/gophercloud/gophercloud/openstack/blockstorage/extensions/volumeactions"
)

func expandBlockStorageV2AttachMode(v string) (volumeactions.AttachMode, error) {
	var attachMode volumeactions.AttachMode
	var attachError error

	switch v {
	case "":
		attachMode = ""
	case "ro":
		attachMode = volumeactions.ReadOnly
	case "rw":
		attachMode = volumeactions.ReadWrite
	default:
		attachError = fmt.Errorf("Invalid attach_mode specified")
	}

	return attachMode, attachError
}

func blockStorageVolumeAttachV2ParseID(id string) (string, string, error) {
	parts := strings.Split(id, "/")
	if len(parts) < 2 {
		return "", "", fmt.Errorf("Unable to determine openstack_blockstorage_volume_attach_v2 ID")
	}

	return parts[0], parts[1], nil
}
