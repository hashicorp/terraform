package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/service/servicediscovery"
	"github.com/hashicorp/terraform/terraform"
)

func resourceAwsServiceDiscoveryServiceMigrateState(
	v int, is *terraform.InstanceState, meta interface{}) (*terraform.InstanceState, error) {
	switch v {
	case 0:
		log.Println("[INFO] Found AWS ServiceDiscovery Service State v0; migrating to v1")
		return migrateServiceDiscoveryServiceStateV0toV1(is)
	default:
		return is, fmt.Errorf("Unexpected schema version: %d", v)
	}
}

func migrateServiceDiscoveryServiceStateV0toV1(is *terraform.InstanceState) (*terraform.InstanceState, error) {
	if is.Empty() {
		log.Println("[DEBUG] Empty InstanceState; nothing to migrate.")
		return is, nil
	}

	log.Printf("[DEBUG] Attributes before migration: %#v", is.Attributes)

	if v, ok := is.Attributes["dns_config.0.routing_policy"]; !ok && v == "" {
		is.Attributes["dns_config.0.routing_policy"] = servicediscovery.RoutingPolicyMultivalue
	}

	log.Printf("[DEBUG] Attributes after migration: %#v", is.Attributes)
	return is, nil
}
