package aws

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/terraform"
)

func resourceAwsRoute53RecordMigrateState(
	v int, is *terraform.InstanceState, meta interface{}) (*terraform.InstanceState, error) {
	switch v {
	case 0:
		log.Println("[INFO] Found AWS Route 53 Record State v0; migrating to v1")
		return migrateR53RecordStateV0toV1(is)
	default:
		return is, fmt.Errorf("Unexpected schema version: %d", v)
	}
}

func migrateR53RecordStateV0toV1(is *terraform.InstanceState) (*terraform.InstanceState, error) {
	if is.Empty() {
		log.Println("[DEBUG] Empty InstanceState; nothing to migrate.")
		return is, nil
	}
	log.Printf("[DEBUG] Attributes before migration: %#v", is.Attributes)
	if is.Attributes["weight"] != "" && is.Attributes["weight"] != "-1" {
		is.Attributes["weighted_routing_policy.#"] = "1"
		key := fmt.Sprintf("weighted_routing_policy.0.weight")
		is.Attributes[key] = is.Attributes["weight"]
	}
	if is.Attributes["failover"] != "" {
		is.Attributes["failover_routing_policy.#"] = "1"
		key := fmt.Sprintf("failover_routing_policy.0.type")
		is.Attributes[key] = is.Attributes["failover"]
	}
	delete(is.Attributes, "weight")
	delete(is.Attributes, "failover")
	log.Printf("[DEBUG] Attributes after migration: %#v", is.Attributes)
	return is, nil
}
