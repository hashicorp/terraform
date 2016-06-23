package vsphere

import (
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform/terraform"
)

func resourceVSphereVirtualMachineMigrateState(
	v int, is *terraform.InstanceState, meta interface{}) (*terraform.InstanceState, error) {
	if is.Empty() {
		log.Println("[DEBUG] Empty InstanceState; nothing to migrate.")
		return is, nil
	}

	switch v {
	case 0:
		log.Println("[INFO] Found Compute Instance State v0; migrating to v1")
		is, err := migrateVSphereVirtualMachineStateV0toV1(is)
		if err != nil {
			return is, err
		}
		return is, nil
	default:
		return is, fmt.Errorf("Unexpected schema version: %d", v)
	}
}

func migrateVSphereVirtualMachineStateV0toV1(is *terraform.InstanceState) (*terraform.InstanceState, error) {
	if is.Empty() || is.Attributes == nil {
		log.Println("[DEBUG] Empty VSphere Virtual Machine State; nothing to migrate.")
		return is, nil
	}

	log.Printf("[DEBUG] Attributes before migration: %#v", is.Attributes)

	if is.Attributes["skip_customization"] == "" {
		is.Attributes["skip_customization"] = "false"
	}

	if is.Attributes["enable_disk_uuid"] == "" {
		is.Attributes["enable_disk_uuid"] = "false"
	}

	for k, _ := range is.Attributes {
		if strings.HasPrefix(k, "disk.") && strings.HasSuffix(k, ".size") {
			diskParts := strings.Split(k, ".")
			if len(diskParts) != 3 {
				continue
			}
			s := strings.Join([]string{diskParts[0], diskParts[1], "controller_type"}, ".")
			if _, ok := is.Attributes[s]; !ok {
				is.Attributes[s] = "scsi"
			}
		}
	}

	log.Printf("[DEBUG] Attributes after migration: %#v", is.Attributes)
	return is, nil
}
