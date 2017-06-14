package aws

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/terraform"
)

func resourceAwsInstanceMigrateState(
	v int, is *terraform.InstanceState, meta interface{}) (*terraform.InstanceState, error) {
	switch v {
	case 0:
		log.Println("[INFO] Found AWS Instance State v0; migrating to v1")
		return migrateAwsInstanceStateV0toV1(is)
	default:
		return is, fmt.Errorf("Unexpected schema version: %d", v)
	}
}

func migrateAwsInstanceStateV0toV1(is *terraform.InstanceState) (*terraform.InstanceState, error) {
	if is.Empty() || is.Attributes == nil {
		log.Println("[DEBUG] Empty InstanceState; nothing to migrate.")
		return is, nil
	}

	log.Printf("[DEBUG] Attributes before migration: %#v", is.Attributes)

	// Delete old count
	delete(is.Attributes, "block_device.#")

	oldBds, err := readV0BlockDevices(is)
	if err != nil {
		return is, err
	}
	// seed count fields for new types
	is.Attributes["ebs_block_device.#"] = "0"
	is.Attributes["ephemeral_block_device.#"] = "0"
	// depending on if state was v0.3.7 or an earlier version, it might have
	// root_block_device defined already
	if _, ok := is.Attributes["root_block_device.#"]; !ok {
		is.Attributes["root_block_device.#"] = "0"
	}
	for _, oldBd := range oldBds {
		if err := writeV1BlockDevice(is, oldBd); err != nil {
			return is, err
		}
	}
	log.Printf("[DEBUG] Attributes after migration: %#v", is.Attributes)
	return is, nil
}

func readV0BlockDevices(is *terraform.InstanceState) (map[string]map[string]string, error) {
	oldBds := make(map[string]map[string]string)
	for k, v := range is.Attributes {
		if !strings.HasPrefix(k, "block_device.") {
			continue
		}
		path := strings.Split(k, ".")
		if len(path) != 3 {
			return oldBds, fmt.Errorf("Found unexpected block_device field: %#v", k)
		}
		hashcode, attribute := path[1], path[2]
		oldBd, ok := oldBds[hashcode]
		if !ok {
			oldBd = make(map[string]string)
			oldBds[hashcode] = oldBd
		}
		oldBd[attribute] = v
		delete(is.Attributes, k)
	}
	return oldBds, nil
}

func writeV1BlockDevice(
	is *terraform.InstanceState, oldBd map[string]string) error {
	code := hashcode.String(oldBd["device_name"])
	bdType := "ebs_block_device"
	if vn, ok := oldBd["virtual_name"]; ok && strings.HasPrefix(vn, "ephemeral") {
		bdType = "ephemeral_block_device"
	} else if dn, ok := oldBd["device_name"]; ok && dn == "/dev/sda1" {
		bdType = "root_block_device"
	}

	switch bdType {
	case "ebs_block_device":
		delete(oldBd, "virtual_name")
	case "root_block_device":
		delete(oldBd, "virtual_name")
		delete(oldBd, "encrypted")
		delete(oldBd, "snapshot_id")
	case "ephemeral_block_device":
		delete(oldBd, "delete_on_termination")
		delete(oldBd, "encrypted")
		delete(oldBd, "iops")
		delete(oldBd, "volume_size")
		delete(oldBd, "volume_type")
	}
	for attr, val := range oldBd {
		attrKey := fmt.Sprintf("%s.%d.%s", bdType, code, attr)
		is.Attributes[attrKey] = val
	}

	countAttr := fmt.Sprintf("%s.#", bdType)
	count, _ := strconv.Atoi(is.Attributes[countAttr])
	is.Attributes[countAttr] = strconv.Itoa(count + 1)
	return nil
}
