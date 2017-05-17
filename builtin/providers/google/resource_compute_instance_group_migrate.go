package google

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

func resourceComputeInstanceGroupMigrateState(
	v int, is *terraform.InstanceState, meta interface{}) (*terraform.InstanceState, error) {
	if is.Empty() {
		log.Println("[DEBUG] Empty InstanceState; nothing to migrate.")
		return is, nil
	}

	switch v {
	case 0:
		log.Println("[INFO] Found Compute Instance Group State v0; migrating to v1")
		is, err := migrateInstanceGroupStateV0toV1(is)
		if err != nil {
			return is, err
		}
		return is, nil
	default:
		return is, fmt.Errorf("Unexpected schema version: %d", v)
	}
}

func migrateInstanceGroupStateV0toV1(is *terraform.InstanceState) (*terraform.InstanceState, error) {
	log.Printf("[DEBUG] Attributes before migration: %#v", is.Attributes)

	newInstances := []string{}

	for k, v := range is.Attributes {
		if !strings.HasPrefix(k, "instances.") {
			continue
		}

		if k == "instances.#" {
			continue
		}

		// Key is now of the form instances.%d
		kParts := strings.Split(k, ".")

		// Sanity check: two parts should be there and <N> should be a number
		badFormat := false
		if len(kParts) != 2 {
			badFormat = true
		} else if _, err := strconv.Atoi(kParts[1]); err != nil {
			badFormat = true
		}

		if badFormat {
			return is, fmt.Errorf("migration error: found instances key in unexpected format: %s", k)
		}

		newInstances = append(newInstances, v)
		delete(is.Attributes, k)
	}

	for _, v := range newInstances {
		hash := schema.HashString(v)
		newKey := fmt.Sprintf("instances.%d", hash)
		is.Attributes[newKey] = v
	}

	log.Printf("[DEBUG] Attributes after migration: %#v", is.Attributes)
	return is, nil
}
