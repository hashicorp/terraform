package aws

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/terraform"
)

func resourceAwsSqsQueuePolicyMigrateState(
	v int, is *terraform.InstanceState, meta interface{}) (*terraform.InstanceState, error) {
	switch v {
	case 0:
		log.Println("[INFO] Found AWS SQS Query Policy State v0; migrating to v1")
		return migrateSqsQueuePolicyStateV0toV1(is)
	default:
		return is, fmt.Errorf("Unexpected schema version: %d", v)
	}
}

func migrateSqsQueuePolicyStateV0toV1(is *terraform.InstanceState) (*terraform.InstanceState, error) {

	if is.Empty() {
		log.Println("[DEBUG] Empty InstanceState; nothing to migrate.")

		return is, nil
	}

	log.Printf("[DEBUG] Attributes before migration: %#v", is.Attributes)

	is.Attributes["id"] = is.Attributes["queue_url"]
	is.ID = is.Attributes["queue_url"]

	log.Printf("[DEBUG] Attributes after migration: %#v, new id: %s", is.Attributes, is.Attributes["queue_url"])

	return is, nil

}
