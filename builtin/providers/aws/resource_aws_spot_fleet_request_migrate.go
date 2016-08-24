package aws

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/terraform"
)

func resourceAwsSpotFleetRequestMigrateState(
	v int, is *terraform.InstanceState, meta interface{}) (*terraform.InstanceState, error) {
	switch v {
	case 0:
		log.Println("[INFO] Found AWS Spot Fleet Request State v0; migrating to v1")
		return migrateSpotFleetRequestV0toV1(is)
	default:
		return is, fmt.Errorf("Unexpected schema version: %d", v)
	}
}

func migrateSpotFleetRequestV0toV1(is *terraform.InstanceState) (*terraform.InstanceState, error) {
	if is.Empty() {
		log.Println("[DEBUG] Empty Spot Fleet Request State; nothing to migrate.")
		return is, nil
	}

	log.Printf("[DEBUG] Attributes before migration: %#v", is.Attributes)

	is.Attributes["associate_public_ip_address"] = "false"

	log.Printf("[DEBUG] Attributes after migration: %#v", is.Attributes)
	return is, nil
}
