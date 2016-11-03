package aws

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/terraform"
)

func resourceAwsDirectoryServiceDirectoryMigrateState(
	v int, is *terraform.InstanceState, meta interface{}) (*terraform.InstanceState, error) {

	switch v {
	case 0:
		log.Println("[INFO] Found AWS Directory Service Directory State v0; migrating to v1")
		return migrateDirectoryServiceStateV0toV1(is)
	default:
		return is, fmt.Errorf("Unexpected schema version: %d", v)
	}
}

func migrateDirectoryServiceStateV0toV1(is *terraform.InstanceState) (*terraform.InstanceState, error) {
	if is.Empty() {
		log.Println("[DEBUG] Empty InstanceState; nothing to migrate.")
		return is, nil
	}

	log.Printf("[DEBUG] Attributes before migration %#v", is.Attributes)

	// Replace password with password hash
	is.Attributes["password"] = directoryServicePasswordHashSha256(is.Attributes["password"])

	log.Printf("[DEBUG] Attributes after migration: %#v", is.Attributes)
	return is, nil
}
