package aws

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/terraform"
)

func resourceAwsSsmAssociationMigrateState(
	v int, is *terraform.InstanceState, meta interface{}) (*terraform.InstanceState, error) {
	switch v {
	case 0:
		log.Println("[INFO] Found AWS SSM Association State v0; migrating to v1")
		return migrateSsmAssociationStateV0toV1(is)
	default:
		return is, fmt.Errorf("Unexpected schema version: %d", v)
	}
}

func migrateSsmAssociationStateV0toV1(is *terraform.InstanceState) (*terraform.InstanceState, error) {

	if is.Empty() {
		log.Println("[DEBUG] Empty InstanceState; nothing to migrate.")

		return is, nil
	}

	log.Printf("[DEBUG] Attributes before migration: %#v", is.Attributes)

	is.Attributes["id"] = is.Attributes["association_id"]
	is.ID = is.Attributes["association_id"]

	log.Printf("[DEBUG] Attributes after migration: %#v, new id: %s", is.Attributes, is.Attributes["association_id"])

	return is, nil

}
